# TICK-1048 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)

### Cell base, unbudgeted count: a6, pk = 3 (broad 50% value, 6% ACL) (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=25882.00..25882.01 rows=1 width=8) (actual time=14.135..20.784 rows=1 loops=1)
  Buffers: shared hit=13326
  ->  Limit  (cost=1001.12..25873.25 rows=700 width=4) (actual time=0.526..20.621 rows=1001 loops=1)
        Buffers: shared hit=13326
        ->  Result  (cost=1001.12..25873.25 rows=700 width=4) (actual time=0.524..20.472 rows=1001 loops=1)
              Buffers: shared hit=13326
              ->  Append  (cost=1001.12..25866.25 rows=700 width=0) (actual time=0.524..20.251 rows=1001 loops=1)
                    Buffers: shared hit=13326
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1001.12..25490.67 rows=696 width=0) (actual time=0.523..20.063 rows=1001 loops=1)
                          Buffers: shared hit=13326
                          ->  Limit  (cost=1001.12..25483.71 rows=696 width=4) (actual time=0.522..19.882 rows=1001 loops=1)
                                Buffers: shared hit=13326
                                ->  Gather  (cost=1001.12..25483.71 rows=696 width=4) (actual time=0.521..19.732 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=13326
                                      ->  Result  (cost=1.11..24414.11 rows=409 width=4) (actual time=0.226..9.871 rows=642 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=13326
                                            ->  Nested Loop Semi Join  (cost=1.11..24414.11 rows=409 width=0) (actual time=0.215..9.746 rows=642 loops=2)
                                                  Buffers: shared hit=13326
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.27 rows=3582 width=32) (actual time=0.059..0.600 rows=1330 loops=2)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1396
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6.61 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=2661)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                        Heap Fetches: 1283
                                                        Buffers: shared hit=11930
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..190.41 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..190.39 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..190.39 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..190.39 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=16 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..148.00 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..147.99 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..147.99 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..147.99 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.67 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.66 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.66 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.66 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.25 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.034 ms
Execution Time: 20.968 ms
```

### Cell base, unbudgeted count: a6, three broad fields (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10835.97..10835.98 rows=1 width=8) (actual time=120.501..133.855 rows=1 loops=1)
  Buffers: shared hit=62885
  ->  Limit  (cost=8666.49..10833.77 rows=176 width=4) (actual time=66.724..133.603 rows=1001 loops=1)
        Buffers: shared hit=62885
        ->  Result  (cost=8666.49..10833.77 rows=176 width=4) (actual time=66.722..133.458 rows=1001 loops=1)
              Buffers: shared hit=62885
              ->  Append  (cost=8666.49..10832.01 rows=176 width=0) (actual time=66.721..133.266 rows=1001 loops=1)
                    Buffers: shared hit=62885
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10454.32 rows=173 width=0) (actual time=66.719..133.095 rows=1001 loops=1)
                          Buffers: shared hit=62885
                          ->  Limit  (cost=8666.49..10452.59 rows=173 width=4) (actual time=66.718..132.911 rows=1001 loops=1)
                                Buffers: shared hit=62885
                                ->  Gather  (cost=8666.49..10452.59 rows=173 width=4) (actual time=66.717..132.763 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=62885
                                      ->  Result  (cost=7666.49..9435.29 rows=102 width=4) (actual time=63.421..116.444 rows=841 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=62885
                                            ->  Nested Loop Semi Join  (cost=7666.49..9435.29 rows=102 width=0) (actual time=63.412..116.253 rows=841 loops=2)
                                                  Buffers: shared hit=62885
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8915.33 rows=444 width=64) (actual time=63.353..97.238 rows=2089 loops=2)
                                                        Buffers: shared hit=46172
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8372.64 rows=889 width=48) (actual time=63.305..73.457 rows=4180 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=12727
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.27 rows=3582 width=32) (actual time=0.070..3.428 rows=8284 loops=2)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=8574
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=62.458..62.459 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=0.021..31.770 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=8361)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v.record_id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=33445
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.43..1.16 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=4178)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v.record_id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=16713
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..192.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..192.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..192.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..192.47 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..191.84 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..190.24 rows=2 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=16 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_4.record_id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..149.31 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..149.30 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..149.30 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..149.30 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..148.67 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..147.87 rows=1 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_7.record_id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_8.record_id) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..35.02 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..35.01 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..35.01 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..35.01 rows=1 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.83 rows=1 width=80) (never executed)
                                                  Join Filter: (r_3.id = v_9.record_id)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.21 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                  ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_11.record_id) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                        Filter: value
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.43..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_9.record_id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.648 ms
Execution Time: 134.040 ms
```

### Cell base, unbudgeted count: a6, pk = 20 (partly correlated, 500 visible) (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=7830.60..7830.61 rows=1 width=8) (actual time=24.692..24.702 rows=1 loops=1)
  Buffers: shared hit=17943
  ->  Limit  (cost=6727.40..7830.03 rows=45 width=4) (actual time=15.383..24.644 rows=494 loops=1)
        Buffers: shared hit=17943
        ->  Result  (cost=6727.40..7830.03 rows=45 width=4) (actual time=15.382..24.586 rows=494 loops=1)
              Buffers: shared hit=17943
              ->  Append  (cost=6727.40..7829.58 rows=45 width=0) (actual time=15.381..24.513 rows=494 loops=1)
                    Buffers: shared hit=17943
                    ->  Subquery Scan on "*SELECT* 1"  (cost=6727.40..7457.29 rows=42 width=0) (actual time=15.380..24.369 rows=494 loops=1)
                          Buffers: shared hit=17929
                          ->  Limit  (cost=6727.40..7456.87 rows=42 width=4) (actual time=15.378..24.308 rows=494 loops=1)
                                Buffers: shared hit=17929
                                ->  Result  (cost=6727.40..7456.87 rows=42 width=4) (actual time=15.377..24.256 rows=494 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=17929
                                      ->  Hash Semi Join  (cost=6727.40..7456.87 rows=42 width=0) (actual time=15.369..24.185 rows=494 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=17929
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..713.35 rows=6090 width=32) (actual time=0.039..5.462 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13045
                                            ->  Hash  (cost=6657.23..6657.23 rows=5569 width=32) (actual time=15.295..15.296 rows=21487 loops=1)
                                                  Buckets: 32768 (originally 8192)  Batches: 1 (originally 1)  Memory Usage: 1599kB
                                                  Buffers: shared hit=4884
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6657.23 rows=5569 width=32) (actual time=0.024..11.722 rows=21487 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '20'::double precision))
                                                        Heap Fetches: 21487
                                                        Buffers: shared hit=4884
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..190.40 rows=1 width=0) (actual time=0.037..0.038 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..190.39 rows=1 width=4) (actual time=0.036..0.037 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..190.39 rows=1 width=4) (actual time=0.035..0.035 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..190.39 rows=1 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=16 width=32) (actual time=0.020..0.021 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '20'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..148.00 rows=1 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..147.99 rows=1 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..147.99 rows=1 width=4) (actual time=0.014..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..147.99 rows=1 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=12 width=32) (actual time=0.012..0.012 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '20'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.67 rows=1 width=0) (actual time=0.024..0.027 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.66 rows=1 width=4) (actual time=0.023..0.025 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.66 rows=1 width=4) (actual time=0.023..0.025 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.66 rows=1 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.021..0.023 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.021..0.022 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.25 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '20'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.425 ms
Execution Time: 24.827 ms
```

### Cell base, unbudgeted count: a50, pk = 3 (broad 50% value, 50% ACL) (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=27627.32..27627.33 rows=1 width=8) (actual time=48.747..55.052 rows=1 loops=1)
  Buffers: shared hit=9413
  ->  Limit  (cost=26203.75..27614.81 rows=1001 width=4) (actual time=46.755..54.959 rows=1001 loops=1)
        Buffers: shared hit=9413
        ->  Result  (cost=26203.75..27619.04 rows=1004 width=4) (actual time=46.753..54.858 rows=1001 loops=1)
              Buffers: shared hit=9413
              ->  Append  (cost=26203.75..27609.00 rows=1004 width=0) (actual time=46.752..54.724 rows=1001 loops=1)
                    Buffers: shared hit=9413
                    ->  Subquery Scan on "*SELECT* 1"  (cost=26203.75..27275.20 rows=1001 width=0) (actual time=46.751..54.616 rows=1001 loops=1)
                          Buffers: shared hit=9413
                          ->  Limit  (cost=26203.75..27265.19 rows=1001 width=4) (actual time=46.749..54.495 rows=1001 loops=1)
                                Buffers: shared hit=9413
                                ->  Gather  (cost=26203.75..32267.02 rows=5718 width=4) (actual time=46.748..54.404 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=9413
                                      ->  Result  (cost=25203.75..30695.22 rows=2382 width=4) (actual time=42.902..44.697 rows=707 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=9413
                                            ->  Parallel Hash Semi Join  (cost=25203.75..30695.22 rows=2382 width=0) (actual time=42.894..44.601 rows=707 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=9413
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5407.82 rows=20842 width=32) (actual time=0.259..1.037 rows=1434 loops=3)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2209
                                                  ->  Parallel Hash  (cost=24723.69..24723.69 rows=38360 width=32) (actual time=42.182..42.183 rows=51950 loops=3)
                                                        Buckets: 262144 (originally 131072)  Batches: 1 (originally 1)  Memory Usage: 0kB
                                                        Buffers: shared hit=7146
                                                        ->  Parallel Bitmap Heap Scan on rfv_num v  (cost=6978.38..24723.69 rows=38360 width=32) (actual time=9.033..24.726 rows=51950 loops=3)
                                                              Recheck Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                              Heap Blocks: exact=1763
                                                              Buffers: shared hit=7146
                                                              ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..6955.36 rows=92065 width=0) (actual time=11.037..11.037 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                                    Buffers: shared hit=2600
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..160.05 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..160.04 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..160.04 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..160.04 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..134.83 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..134.82 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..134.82 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..134.82 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.562 ms
Execution Time: 55.189 ms
```

### Cell base, unbudgeted count: a6, pk < 5 (every 3 and 4: 78% of the side rows) (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=12439.72..12439.73 rows=1 width=8) (actual time=8.902..13.743 rows=1 loops=1)
  Buffers: shared hit=5181
  ->  Limit  (cost=1000.99..12427.20 rows=1001 width=4) (actual time=0.403..13.599 rows=1001 loops=1)
        Buffers: shared hit=5181
        ->  Result  (cost=1000.99..12564.18 rows=1013 width=4) (actual time=0.402..13.476 rows=1001 loops=1)
              Buffers: shared hit=5181
              ->  Append  (cost=1000.99..12554.05 rows=1013 width=0) (actual time=0.401..13.319 rows=1001 loops=1)
                    Buffers: shared hit=5181
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1000.99..12160.80 rows=1001 width=0) (actual time=0.400..13.180 rows=1001 loops=1)
                          Buffers: shared hit=5181
                          ->  Limit  (cost=1000.99..12150.79 rows=1001 width=4) (actual time=0.399..13.035 rows=1001 loops=1)
                                Buffers: shared hit=5181
                                ->  Gather  (cost=1000.99..27121.16 rows=2345 width=4) (actual time=0.399..12.911 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=5181
                                      ->  Result  (cost=0.99..25886.66 rows=1379 width=4) (actual time=0.105..5.114 rows=558 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=5181
                                            ->  Nested Loop Semi Join  (cost=0.99..25886.66 rows=1379 width=0) (actual time=0.097..5.020 rows=558 loops=2)
                                                  Buffers: shared hit=5181
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.27 rows=3582 width=32) (actual time=0.050..0.278 rows=572 loops=2)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=604
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..7.28 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=1144)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Filter: (value < '5'::double precision)
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=4577
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..199.82 rows=6 width=0) (never executed)
                          ->  Limit  (cost=0.99..199.76 rows=6 width=4) (never executed)
                                ->  Result  (cost=0.99..199.76 rows=6 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..199.76 rows=6 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=16 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_1.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..154.69 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.99..154.64 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.99..154.64 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..154.64 rows=5 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_2.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.67 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.66 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.66 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.66 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.25 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value < '5'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.496 ms
Execution Time: 13.879 ms
```

### Cell base, unbudgeted count: a50, pw = 1 (704k invisible side rows, 100 visible) (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9084.35..9084.36 rows=1 width=8) (actual time=738.053..738.060 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9071.84 rows=1001 width=4) (actual time=4.254..737.871 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9162.46 rows=1011 width=4) (actual time=4.253..737.765 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9152.35 rows=1011 width=0) (actual time=4.252..737.596 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8820.93 rows=1001 width=0) (actual time=4.250..737.425 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8810.92 rows=1001 width=4) (actual time=4.248..737.316 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169537.05 rows=19263 width=4) (actual time=4.247..737.248 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169537.05 rows=19263 width=0) (actual time=4.238..737.141 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.042..51.868 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..158.72 rows=5 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..158.67 rows=5 width=4) (actual time=0.034..0.035 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..158.67 rows=5 width=4) (actual time=0.032..0.033 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..158.67 rows=5 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.019..0.019 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..133.73 rows=4 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..133.69 rows=4 width=4) (actual time=0.014..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..133.69 rows=4 width=4) (actual time=0.013..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..133.69 rows=4 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.010..0.010 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.93 rows=1 width=0) (actual time=0.022..0.025 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.92 rows=1 width=4) (actual time=0.021..0.024 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.92 rows=1 width=4) (actual time=0.021..0.023 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.92 rows=1 width=0) (actual time=0.019..0.021 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.019..0.020 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.018..0.020 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.016..0.017 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.51 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.594 ms
Execution Time: 738.174 ms
```

### Cell base, unbudgeted count: a6, pw = 1 (704k invisible side rows, 100 visible) (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=11661.94..11661.95 rows=1 width=8) (actual time=61.712..71.517 rows=1 loops=1)
  Buffers: shared hit=109385
  ->  Limit  (cost=1001.12..11649.42 rows=1001 width=4) (actual time=0.428..71.470 rows=100 loops=1)
        Buffers: shared hit=109385
        ->  Result  (cost=1001.12..11777.08 rows=1013 width=4) (actual time=0.427..71.440 rows=100 loops=1)
              Buffers: shared hit=109385
              ->  Append  (cost=1001.12..11766.95 rows=1013 width=0) (actual time=0.426..71.399 rows=100 loops=1)
                    Buffers: shared hit=109385
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1001.12..11392.72 rows=1001 width=0) (actual time=0.425..71.271 rows=100 loops=1)
                          Buffers: shared hit=109371
                          ->  Limit  (cost=1001.12..11382.71 rows=1001 width=4) (actual time=0.424..71.214 rows=100 loops=1)
                                Buffers: shared hit=109371
                                ->  Gather  (cost=1001.12..25321.63 rows=2345 width=4) (actual time=0.423..71.191 rows=100 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=109371
                                      ->  Result  (cost=1.11..24087.13 rows=1379 width=4) (actual time=2.658..58.395 rows=50 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=109371
                                            ->  Nested Loop Semi Join  (cost=1.11..24087.13 rows=1379 width=0) (actual time=2.650..58.361 rows=50 loops=2)
                                                  Buffers: shared hit=109371
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.27 rows=3582 width=32) (actual time=0.049..4.454 rows=12028 loops=2)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13048
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6.61 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=24055)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r.id))
                                                        Heap Fetches: 100
                                                        Buffers: shared hit=96323
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..188.80 rows=6 width=0) (actual time=0.040..0.045 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..188.74 rows=6 width=4) (actual time=0.039..0.042 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..188.74 rows=6 width=4) (actual time=0.037..0.040 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..188.74 rows=6 width=0) (actual time=0.026..0.028 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=16 width=32) (actual time=0.026..0.026 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..146.67 rows=5 width=0) (actual time=0.014..0.017 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..146.62 rows=5 width=4) (actual time=0.013..0.015 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..146.62 rows=5 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..146.62 rows=5 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=12 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.69 rows=1 width=0) (actual time=0.024..0.030 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.68 rows=1 width=4) (actual time=0.024..0.030 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.68 rows=1 width=4) (actual time=0.024..0.028 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.68 rows=1 width=0) (actual time=0.022..0.026 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.021..0.025 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.021..0.024 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.019..0.021 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.017..0.019 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.27 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.539 ms
Execution Time: 71.656 ms
```

### Cell spill, unbudgeted count: a50, pk = 3 (max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=18932.44..18932.45 rows=1 width=8) (actual time=112.848..112.855 rows=1 loops=1)
  Buffers: shared hit=109377, temp read=3109 written=4402
  I/O Timings: temp read=4.283 write=6.012
  ->  Limit  (cost=13196.53..18919.93 rows=1001 width=4) (actual time=111.623..112.785 rows=1001 loops=1)
        Buffers: shared hit=109377, temp read=3109 written=4402
        I/O Timings: temp read=4.283 write=6.012
        ->  Result  (cost=13196.53..18937.08 rows=1004 width=4) (actual time=111.623..112.712 rows=1001 loops=1)
              Buffers: shared hit=109377, temp read=3109 written=4402
              I/O Timings: temp read=4.283 write=6.012
              ->  Append  (cost=13196.53..18927.04 rows=1004 width=0) (actual time=111.622..112.620 rows=1001 loops=1)
                    Buffers: shared hit=109377, temp read=3109 written=4402
                    I/O Timings: temp read=4.283 write=6.012
                    ->  Subquery Scan on "*SELECT* 1"  (cost=13196.53..18593.24 rows=1001 width=0) (actual time=111.621..112.554 rows=1001 loops=1)
                          Buffers: shared hit=109377, temp read=3109 written=4402
                          I/O Timings: temp read=4.283 write=6.012
                          ->  Limit  (cost=13196.53..18583.23 rows=1001 width=4) (actual time=111.620..112.485 rows=1001 loops=1)
                                Buffers: shared hit=109377, temp read=3109 written=4402
                                I/O Timings: temp read=4.283 write=6.012
                                ->  Result  (cost=13196.53..43966.94 rows=5718 width=4) (actual time=111.619..112.429 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=109377, temp read=3109 written=4402
                                      I/O Timings: temp read=4.283 write=6.012
                                      ->  Merge Semi Join  (cost=13196.53..43966.94 rows=5718 width=0) (actual time=111.614..112.347 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=109377, temp read=3109 written=4402
                                            I/O Timings: temp read=4.283 write=6.012
                                            ->  Sort  (cost=13194.72..13319.77 rows=50020 width=32) (actual time=111.571..111.730 rows=2026 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: external merge  Disk: 8296kB
                                                  Buffers: shared hit=109306, temp read=3109 written=4402
                                                  I/O Timings: temp read=4.283 write=6.012
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.029..31.714 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109306
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..30236.02 rows=92065 width=32) (actual time=0.036..0.296 rows=1126 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 1126
                                                  Buffers: shared hit=71
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..160.05 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..160.04 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..160.04 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..160.04 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..134.83 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..134.82 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..134.82 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..134.82 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 0.921 ms
Execution Time: 113.789 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 (max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9084.35..9084.36 rows=1 width=8) (actual time=631.521..631.529 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9071.84 rows=1001 width=4) (actual time=3.698..631.364 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9162.46 rows=1011 width=4) (actual time=3.697..631.270 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9152.35 rows=1011 width=0) (actual time=3.696..631.194 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8820.93 rows=1001 width=0) (actual time=3.695..631.058 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8810.92 rows=1001 width=4) (actual time=3.694..630.992 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169537.05 rows=19263 width=4) (actual time=3.694..630.947 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169537.05 rows=19263 width=0) (actual time=3.687..630.864 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.039..43.697 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..158.72 rows=5 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..158.67 rows=5 width=4) (actual time=0.032..0.033 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..158.67 rows=5 width=4) (actual time=0.030..0.031 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..158.67 rows=5 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..133.73 rows=4 width=0) (actual time=0.013..0.015 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..133.69 rows=4 width=4) (actual time=0.013..0.014 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..133.69 rows=4 width=4) (actual time=0.013..0.013 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..133.69 rows=4 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.010..0.011 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.93 rows=1 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.92 rows=1 width=4) (actual time=0.022..0.023 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.92 rows=1 width=4) (actual time=0.021..0.022 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.92 rows=1 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.018..0.020 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.018..0.019 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.016..0.016 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.51 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.152 ms
Execution Time: 631.632 ms
```

### Cell spill, unbudgeted count: a50, pk = 3 (enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=329885.23..329885.24 rows=1 width=8) (actual time=159.922..159.931 rows=1 loops=1)
  Buffers: shared hit=109377, temp read=3109 written=4402
  I/O Timings: temp read=4.505 write=7.901
  ->  Limit  (cost=13196.53..329872.72 rows=1001 width=4) (actual time=158.341..159.845 rows=1001 loops=1)
        Buffers: shared hit=109377, temp read=3109 written=4402
        I/O Timings: temp read=4.505 write=7.901
        ->  Result  (cost=13196.53..330821.80 rows=1004 width=4) (actual time=158.339..159.751 rows=1001 loops=1)
              Buffers: shared hit=109377, temp read=3109 written=4402
              I/O Timings: temp read=4.505 write=7.901
              ->  Append  (cost=13196.53..330811.76 rows=1004 width=0) (actual time=158.338..159.630 rows=1001 loops=1)
                    Buffers: shared hit=109377, temp read=3109 written=4402
                    I/O Timings: temp read=4.505 write=7.901
                    ->  Subquery Scan on "*SELECT* 1"  (cost=13196.53..18593.24 rows=1001 width=0) (actual time=158.337..159.545 rows=1001 loops=1)
                          Buffers: shared hit=109377, temp read=3109 written=4402
                          I/O Timings: temp read=4.505 write=7.901
                          ->  Limit  (cost=13196.53..18583.23 rows=1001 width=4) (actual time=158.336..159.454 rows=1001 loops=1)
                                Buffers: shared hit=109377, temp read=3109 written=4402
                                I/O Timings: temp read=4.505 write=7.901
                                ->  Result  (cost=13196.53..43966.94 rows=5718 width=4) (actual time=158.336..159.381 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=109377, temp read=3109 written=4402
                                      I/O Timings: temp read=4.505 write=7.901
                                      ->  Merge Semi Join  (cost=13196.53..43966.94 rows=5718 width=0) (actual time=158.328..159.272 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=109377, temp read=3109 written=4402
                                            I/O Timings: temp read=4.505 write=7.901
                                            ->  Sort  (cost=13194.72..13319.77 rows=50020 width=32) (actual time=158.273..158.481 rows=2026 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: external merge  Disk: 8296kB
                                                  Buffers: shared hit=109306, temp read=3109 written=4402
                                                  I/O Timings: temp read=4.505 write=7.901
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.041..47.437 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109306
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..30236.02 rows=92065 width=32) (actual time=0.047..0.376 rows=1126 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 1126
                                                  Buffers: shared hit=71
                    ->  Subquery Scan on "*SELECT* 2"  (cost=69.76..30535.46 rows=1 width=0) (never executed)
                          ->  Limit  (cost=69.76..30535.45 rows=1 width=4) (never executed)
                                ->  Result  (cost=69.76..30535.45 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Merge Semi Join  (cost=69.76..30535.45 rows=1 width=0) (never executed)
                                            Merge Cond: (r_1.id = v_1.record_id)
                                            ->  Sort  (cost=69.19..69.22 rows=12 width=32) (never executed)
                                                  Sort Key: r_1.id
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..30236.02 rows=92065 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=57.66..30523.36 rows=1 width=0) (never executed)
                          ->  Limit  (cost=57.66..30523.35 rows=1 width=4) (never executed)
                                ->  Result  (cost=57.66..30523.35 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Merge Semi Join  (cost=57.66..30523.35 rows=1 width=0) (never executed)
                                            Merge Cond: (r_2.id = v_2.record_id)
                                            ->  Sort  (cost=57.10..57.12 rows=10 width=32) (never executed)
                                                  Sort Key: r_2.id
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..30236.02 rows=92065 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=32040.77..251154.67 rows=1 width=0) (never executed)
                          ->  Limit  (cost=32040.77..251154.66 rows=1 width=4) (never executed)
                                ->  Result  (cost=32040.77..251154.66 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=32040.77..251154.66 rows=1 width=0) (never executed)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            ->  Hash Semi Join  (cost=23.93..218505.82 rows=1 width=48) (never executed)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  ->  Seq Scan on records r_3  (cost=0.00..218120.60 rows=137635 width=32) (never executed)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Hash  (cost=30236.02..30236.02 rows=92065 width=32) (never executed)
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.55..30236.02 rows=92065 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                        Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.322 ms
Execution Time: 161.165 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 (enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=420254.52..420254.53 rows=1 width=8) (actual time=678.692..678.700 rows=1 loops=1)
  Buffers: shared hit=424092
  ->  Limit  (cost=4121.48..420242.01 rows=1001 width=4) (actual time=0.059..678.687 rows=100 loops=1)
        Buffers: shared hit=424092
        ->  Result  (cost=4121.48..424399.06 rows=1011 width=4) (actual time=0.057..678.677 rows=100 loops=1)
              Buffers: shared hit=424092
              ->  Append  (cost=4121.48..424388.95 rows=1011 width=0) (actual time=0.057..678.664 rows=100 loops=1)
                    Buffers: shared hit=424092
                    ->  Subquery Scan on "*SELECT* 1"  (cost=4121.48..49310.42 rows=1001 width=0) (actual time=0.056..678.551 rows=100 loops=1)
                          Buffers: shared hit=424072
                          ->  Limit  (cost=4121.48..49300.41 rows=1001 width=4) (actual time=0.055..678.538 rows=100 loops=1)
                                Buffers: shared hit=424072
                                ->  Result  (cost=4121.48..873533.84 rows=19263 width=4) (actual time=0.055..678.530 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=424072
                                      ->  Merge Semi Join  (cost=4121.48..873533.84 rows=19263 width=0) (actual time=0.050..678.514 rows=100 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=424072
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..824303.09 rows=50020 width=32) (actual time=0.024..475.107 rows=200376 loops=1)
                                                  Index Cond: (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                  Rows Removed by Filter: 199621
                                                  Buffers: shared hit=404376
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..52179.49 rows=335361 width=32) (actual time=0.020..142.769 rows=702464 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                  Heap Fetches: 702464
                                                  Buffers: shared hit=19696
                    ->  Subquery Scan on "*SELECT* 2"  (cost=49852.69..52216.28 rows=5 width=0) (actual time=0.032..0.034 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49852.69..52216.23 rows=5 width=4) (actual time=0.031..0.033 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49852.69..52216.23 rows=5 width=4) (actual time=0.030..0.031 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49852.69..52216.23 rows=5 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.017..0.017 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=49852.69..52204.21 rows=4 width=0) (actual time=0.010..0.012 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49852.69..52204.17 rows=4 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49852.69..52204.17 rows=4 width=4) (actual time=0.010..0.011 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49852.69..52204.17 rows=4 width=0) (actual time=0.007..0.009 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.007..0.007 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=49876.07..270652.98 rows=1 width=0) (actual time=0.052..0.054 rows=0 loops=1)
                          Buffers: shared hit=12
                          ->  Limit  (cost=49876.07..270652.97 rows=1 width=4) (actual time=0.051..0.052 rows=0 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=49876.07..270652.97 rows=1 width=4) (actual time=0.050..0.052 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=12
                                      ->  Hash Semi Join  (cost=49876.07..270652.97 rows=1 width=0) (actual time=0.049..0.050 rows=0 loops=1)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            Buffers: shared hit=12
                                            ->  Hash Semi Join  (cost=23.93..218505.82 rows=1 width=48) (actual time=0.048..0.049 rows=0 loops=1)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  Buffers: shared hit=12
                                                  ->  Seq Scan on records r_3  (cost=0.00..218120.60 rows=137635 width=32) (actual time=0.029..0.029 rows=1 loops=1)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=6
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                          Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.344 ms
Execution Time: 678.795 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=421722.80..421722.81 rows=1 width=8) (actual time=579.014..579.023 rows=1 loops=1)
  Buffers: shared hit=126400, temp read=7920 written=7920
  I/O Timings: temp read=11.772 write=101.329
  ->  Limit  (cost=49852.69..421710.29 rows=1001 width=4) (actual time=459.710..578.968 rows=100 loops=1)
        Buffers: shared hit=126400, temp read=7920 written=7920
        I/O Timings: temp read=11.772 write=101.329
        ->  Result  (cost=49852.69..425425.15 rows=1011 width=4) (actual time=459.709..578.943 rows=100 loops=1)
              Buffers: shared hit=126400, temp read=7920 written=7920
              I/O Timings: temp read=11.772 write=101.329
              ->  Append  (cost=49852.69..425415.04 rows=1011 width=0) (actual time=459.708..578.912 rows=100 loops=1)
                    Buffers: shared hit=126400, temp read=7920 written=7920
                    I/O Timings: temp read=11.772 write=101.329
                    ->  Subquery Scan on "*SELECT* 1"  (cost=49852.69..50336.52 rows=1001 width=0) (actual time=459.707..578.735 rows=100 loops=1)
                          Buffers: shared hit=126380, temp read=7920 written=7920
                          I/O Timings: temp read=11.772 write=101.329
                          ->  Limit  (cost=49852.69..50326.51 rows=1001 width=4) (actual time=459.706..578.695 rows=100 loops=1)
                                Buffers: shared hit=126380, temp read=7920 written=7920
                                I/O Timings: temp read=11.772 write=101.329
                                ->  Result  (cost=49852.69..58970.66 rows=19263 width=4) (actual time=459.705..578.664 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=126380, temp read=7920 written=7920
                                      I/O Timings: temp read=11.772 write=101.329
                                      ->  Hash Semi Join  (cost=49852.69..58970.66 rows=19263 width=0) (actual time=459.697..578.618 rows=100 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=126380, temp read=7920 written=7920
                                            I/O Timings: temp read=11.772 write=101.329
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.035..47.592 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (actual time=309.004..309.005 rows=702464 loops=1)
                                                  Buckets: 2048 (originally 2048)  Batches: 512 (originally 256)  Memory Usage: 113kB
                                                  Buffers: shared hit=17074, temp written=4235
                                                  I/O Timings: temp write=26.544
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..43367.12 rows=335361 width=32) (actual time=69.077..179.526 rows=702464 loops=1)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                        Rows Removed by Filter: 800000
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=49852.69..52216.28 rows=5 width=0) (actual time=0.051..0.052 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49852.69..52216.23 rows=5 width=4) (actual time=0.050..0.051 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49852.69..52216.23 rows=5 width=4) (actual time=0.048..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49852.69..52216.23 rows=5 width=0) (actual time=0.034..0.034 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.032..0.032 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=49852.69..52204.21 rows=4 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49852.69..52204.17 rows=4 width=4) (actual time=0.017..0.018 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49852.69..52204.17 rows=4 width=4) (actual time=0.017..0.017 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49852.69..52204.17 rows=4 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.013..0.013 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=49876.07..270652.98 rows=1 width=0) (actual time=0.072..0.074 rows=0 loops=1)
                          Buffers: shared hit=12
                          ->  Limit  (cost=49876.07..270652.97 rows=1 width=4) (actual time=0.072..0.073 rows=0 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=49876.07..270652.97 rows=1 width=4) (actual time=0.071..0.073 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=12
                                      ->  Hash Semi Join  (cost=49876.07..270652.97 rows=1 width=0) (actual time=0.070..0.071 rows=0 loops=1)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            Buffers: shared hit=12
                                            ->  Hash Semi Join  (cost=23.93..218505.82 rows=1 width=48) (actual time=0.069..0.070 rows=0 loops=1)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  Buffers: shared hit=12
                                                  ->  Seq Scan on records r_3  (cost=0.00..218120.60 rows=137635 width=32) (actual time=0.045..0.045 rows=1 loops=1)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=6
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.016..0.017 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                          Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.805 ms
Execution Time: 579.125 ms
```

### Cell spill, unbudgeted count: a50, pk < 5 (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=425681.04..425681.05 rows=1 width=8) (actual time=357.031..357.041 rows=1 loops=1)
  Buffers: shared hit=126380, temp read=10 written=2949
  I/O Timings: temp read=0.024 write=22.159
  ->  Limit  (cost=50638.38..425668.53 rows=1001 width=4) (actual time=264.170..356.898 rows=1001 loops=1)
        Buffers: shared hit=126380, temp read=10 written=2949
        I/O Timings: temp read=0.024 write=22.159
        ->  Result  (cost=50638.38..429415.09 rows=1011 width=4) (actual time=264.169..356.779 rows=1001 loops=1)
              Buffers: shared hit=126380, temp read=10 written=2949
              I/O Timings: temp read=0.024 write=22.159
              ->  Append  (cost=50638.38..429404.98 rows=1011 width=0) (actual time=264.168..356.604 rows=1001 loops=1)
                    Buffers: shared hit=126380, temp read=10 written=2949
                    I/O Timings: temp read=0.024 write=22.159
                    ->  Subquery Scan on "*SELECT* 1"  (cost=50638.38..51135.40 rows=1001 width=0) (actual time=264.167..356.471 rows=1001 loops=1)
                          Buffers: shared hit=126380, temp read=10 written=2949
                          I/O Timings: temp read=0.024 write=22.159
                          ->  Limit  (cost=50638.38..51125.39 rows=1001 width=4) (actual time=264.166..356.323 rows=1001 loops=1)
                                Buffers: shared hit=126380, temp read=10 written=2949
                                I/O Timings: temp read=0.024 write=22.159
                                ->  Result  (cost=50638.38..60010.27 rows=19263 width=4) (actual time=264.165..356.202 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=126380, temp read=10 written=2949
                                      I/O Timings: temp read=0.024 write=22.159
                                      ->  Hash Semi Join  (cost=50638.38..60010.27 rows=19263 width=0) (actual time=264.160..356.037 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=126380, temp read=10 written=2949
                                            I/O Timings: temp read=0.024 write=22.159
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.038..49.994 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=262.029..262.030 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=13.848
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..43367.12 rows=375976 width=32) (actual time=43.457..192.295 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=50638.38..53279.97 rows=5 width=0) (never executed)
                          ->  Limit  (cost=50638.38..53279.92 rows=5 width=4) (never executed)
                                ->  Result  (cost=50638.38..53279.92 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=50638.38..53279.92 rows=5 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=375976 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=50638.38..53267.89 rows=4 width=0) (never executed)
                          ->  Limit  (cost=50638.38..53267.85 rows=4 width=4) (never executed)
                                ->  Result  (cost=50638.38..53267.85 rows=4 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=50638.38..53267.85 rows=4 width=0) (never executed)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=375976 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=50661.76..271716.67 rows=1 width=0) (never executed)
                          ->  Limit  (cost=50661.76..271716.66 rows=1 width=4) (never executed)
                                ->  Result  (cost=50661.76..271716.66 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=50661.76..271716.66 rows=1 width=0) (never executed)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            ->  Hash Semi Join  (cost=23.93..218505.82 rows=1 width=48) (never executed)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  ->  Seq Scan on records r_3  (cost=0.00..218120.60 rows=137635 width=32) (never executed)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=375976 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.105 ms
Execution Time: 372.461 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 and pk < 5 (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=630572.26..630572.27 rows=1 width=8) (actual time=1381.700..1381.721 rows=1 loops=1)
  Buffers: shared hit=194696, temp read=8687 written=16288
  I/O Timings: temp read=11.512 write=95.268
  ->  Limit  (cost=100490.51..630559.75 rows=1001 width=4) (actual time=706.526..1381.700 rows=100 loops=1)
        Buffers: shared hit=194696, temp read=8687 written=16288
        I/O Timings: temp read=11.512 write=95.268
        ->  Result  (cost=100490.51..635855.15 rows=1011 width=4) (actual time=706.525..1381.686 rows=100 loops=1)
              Buffers: shared hit=194696, temp read=8687 written=16288
              I/O Timings: temp read=11.512 write=95.268
              ->  Append  (cost=100490.51..635845.04 rows=1011 width=0) (actual time=706.524..1381.668 rows=100 loops=1)
                    Buffers: shared hit=194696, temp read=8687 written=16288
                    I/O Timings: temp read=11.512 write=95.268
                    ->  Subquery Scan on "*SELECT* 1"  (cost=100490.51..101133.97 rows=1001 width=0) (actual time=706.523..720.799 rows=100 loops=1)
                          Buffers: shared hit=143454, temp read=8687 written=10075
                          I/O Timings: temp read=11.512 write=59.489
                          ->  Limit  (cost=100490.51..101123.96 rows=1001 width=4) (actual time=706.522..720.779 rows=100 loops=1)
                                Buffers: shared hit=143454, temp read=8687 written=10075
                                I/O Timings: temp read=11.512 write=59.489
                                ->  Result  (cost=100490.51..112680.39 rows=19263 width=4) (actual time=706.521..720.765 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=143454, temp read=8687 written=10075
                                      I/O Timings: temp read=11.512 write=59.489
                                      ->  Hash Semi Join  (cost=100490.51..112680.39 rows=19263 width=0) (actual time=706.512..720.738 rows=100 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=143454, temp read=8687 written=10075
                                            I/O Timings: temp read=11.512 write=59.489
                                            ->  Hash Semi Join  (cost=49852.68..58970.65 rows=19263 width=48) (actual time=378.974..489.250 rows=100 loops=1)
                                                  Hash Cond: (r.id = v_1.record_id)
                                                  Buffers: shared hit=126380, temp read=7920 written=7920
                                                  I/O Timings: temp read=10.481 write=47.558
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.046..40.308 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109306
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (actual time=289.674..289.676 rows=702464 loops=1)
                                                        Buckets: 2048 (originally 2048)  Batches: 512 (originally 256)  Memory Usage: 113kB
                                                        Buffers: shared hit=17074, temp written=4235
                                                        I/O Timings: temp write=23.943
                                                        ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335361 width=32) (actual time=67.256..169.737 rows=702464 loops=1)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                              Rows Removed by Filter: 800000
                                                              Buffers: shared hit=17074
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=215.869..215.869 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=10.300
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..43367.12 rows=375976 width=32) (actual time=37.876..158.797 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=100490.51..105427.15 rows=5 width=0) (actual time=215.075..215.080 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=2071
                          I/O Timings: temp write=11.413
                          ->  Limit  (cost=100490.51..105427.10 rows=5 width=4) (actual time=215.074..215.078 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=2071
                                I/O Timings: temp write=11.413
                                ->  Result  (cost=100490.51..105427.10 rows=5 width=4) (actual time=215.072..215.076 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=17078, temp written=2071
                                      I/O Timings: temp write=11.413
                                      ->  Hash Semi Join  (cost=100490.51..105427.10 rows=5 width=0) (actual time=215.061..215.064 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=17078, temp written=2071
                                            I/O Timings: temp write=11.413
                                            ->  Hash Semi Join  (cost=49852.68..52216.22 rows=5 width=48) (actual time=0.035..0.037 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_3.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.034..0.034 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=208.430..208.431 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=10.232
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=375976 width=32) (actual time=35.846..154.760 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 3"  (cost=100490.51..105415.07 rows=4 width=0) (actual time=217.828..217.831 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=2071
                          I/O Timings: temp write=11.692
                          ->  Limit  (cost=100490.51..105415.03 rows=4 width=4) (actual time=217.826..217.829 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=2071
                                I/O Timings: temp write=11.692
                                ->  Result  (cost=100490.51..105415.03 rows=4 width=4) (actual time=217.823..217.826 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=17078, temp written=2071
                                      I/O Timings: temp write=11.692
                                      ->  Hash Semi Join  (cost=100490.51..105415.03 rows=4 width=0) (actual time=217.808..217.811 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=17078, temp written=2071
                                            I/O Timings: temp write=11.692
                                            ->  Hash Semi Join  (cost=49852.68..52204.15 rows=4 width=48) (actual time=0.026..0.027 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_5.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.025..0.025 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_5  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=210.512..210.512 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=10.309
                                                  ->  Seq Scan on rfv_num v_4  (cost=0.00..43367.12 rows=375976 width=32) (actual time=35.934..154.605 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 4"  (cost=100513.89..323863.80 rows=1 width=0) (actual time=227.928..227.933 rows=0 loops=1)
                          Buffers: shared hit=17086, temp written=2071
                          I/O Timings: temp write=12.675
                          ->  Limit  (cost=100513.89..323863.79 rows=1 width=4) (actual time=227.925..227.930 rows=0 loops=1)
                                Buffers: shared hit=17086, temp written=2071
                                I/O Timings: temp write=12.675
                                ->  Result  (cost=100513.89..323863.79 rows=1 width=4) (actual time=227.923..227.928 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=17086, temp written=2071
                                      I/O Timings: temp write=12.675
                                      ->  Hash Semi Join  (cost=100513.89..323863.79 rows=1 width=0) (actual time=227.908..227.913 rows=0 loops=1)
                                            Hash Cond: (r_3.id = v_6.record_id)
                                            Buffers: shared hit=17086, temp written=2071
                                            I/O Timings: temp write=12.675
                                            ->  Hash Semi Join  (cost=49876.06..270652.96 rows=1 width=64) (actual time=0.087..0.091 rows=0 loops=1)
                                                  Hash Cond: (r_3.id = v_7.record_id)
                                                  Buffers: shared hit=12
                                                  ->  Hash Semi Join  (cost=23.93..218505.82 rows=1 width=48) (actual time=0.086..0.089 rows=0 loops=1)
                                                        Hash Cond: (r_3.id = acl_share.record_id)
                                                        Buffers: shared hit=12
                                                        ->  Seq Scan on records r_3  (cost=0.00..218120.60 rows=137635 width=32) (actual time=0.054..0.055 rows=1 loops=1)
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                              Rows Removed by Filter: 33
                                                              Buffers: shared hit=6
                                                        ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.022..0.024 rows=0 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                    Buffers: shared hit=6
                                                                    ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                          Buffers: shared hit=6
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.005 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                                Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_7  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=220.591..220.591 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=11.393
                                                  ->  Seq Scan on rfv_num v_6  (cost=0.00..43367.12 rows=375976 width=32) (actual time=35.371..162.063 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.643 ms
Execution Time: 1382.061 ms
```

### Cell spill, unbudgeted count: a50, three broad fields (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=478595.69..478595.70 rows=1 width=8) (actual time=474.428..474.443 rows=1 loops=1)
  Buffers: shared hit=124701, temp read=5005 written=6581
  I/O Timings: temp read=8.255 write=37.651
  ->  Limit  (cost=60987.64..478583.18 rows=1001 width=4) (actual time=257.734..474.330 rows=1001 loops=1)
        Buffers: shared hit=124701, temp read=5005 written=6581
        I/O Timings: temp read=8.255 write=37.651
        ->  Result  (cost=60987.64..479834.71 rows=1004 width=4) (actual time=257.733..474.229 rows=1001 loops=1)
              Buffers: shared hit=124701, temp read=5005 written=6581
              I/O Timings: temp read=8.255 write=37.651
              ->  Append  (cost=60987.64..479824.67 rows=1004 width=0) (actual time=257.732..474.108 rows=1001 loops=1)
                    Buffers: shared hit=124701, temp read=5005 written=6581
                    I/O Timings: temp read=8.255 write=37.651
                    ->  Subquery Scan on "*SELECT* 1"  (cost=60987.64..68125.52 rows=1001 width=0) (actual time=257.730..473.922 rows=1001 loops=1)
                          Buffers: shared hit=124701, temp read=5005 written=6581
                          I/O Timings: temp read=8.255 write=37.651
                          ->  Limit  (cost=60987.64..68115.51 rows=1001 width=4) (actual time=257.729..473.813 rows=1001 loops=1)
                                Buffers: shared hit=124701, temp read=5005 written=6581
                                I/O Timings: temp read=8.255 write=37.651
                                ->  Result  (cost=60987.64..71127.58 rows=1424 width=4) (actual time=257.728..473.724 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=124701, temp read=5005 written=6581
                                      I/O Timings: temp read=8.255 write=37.651
                                      ->  Hash Semi Join  (cost=60987.64..71127.58 rows=1424 width=0) (actual time=257.720..473.589 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=124701, temp read=5005 written=6581
                                            I/O Timings: temp read=8.255 write=37.651
                                            ->  Hash Semi Join  (cost=46998.04..55731.36 rows=1416 width=64) (actual time=166.226..368.508 rows=50162 loops=1)
                                                  Hash Cond: (r.id = v_2.record_id)
                                                  Buffers: shared hit=120577, temp read=4934 written=4934
                                                  I/O Timings: temp read=8.100 write=27.453
                                                  ->  Hash Semi Join  (cost=32017.38..39253.93 rows=5718 width=48) (actual time=70.454..212.313 rows=100217 loops=1)
                                                        Hash Cond: (r.id = v_1.record_id)
                                                        Buffers: shared hit=116453, temp read=2728 written=2728
                                                        I/O Timings: temp read=4.496 write=15.517
                                                        ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.047..44.426 rows=200379 loops=1)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=109306
                                                        ->  Hash  (cost=30236.02..30236.02 rows=92065 width=32) (actual time=69.128..69.129 rows=155850 loops=1)
                                                              Buckets: 2048 (originally 2048)  Batches: 128 (originally 64)  Memory Usage: 113kB
                                                              Buffers: shared hit=7147, temp written=899
                                                              I/O Timings: temp write=5.647
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..30236.02 rows=92065 width=32) (actual time=0.021..40.310 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                                    Heap Fetches: 155850
                                                                    Buffers: shared hit=7147
                                                  ->  Hash  (cost=11124.00..11124.00 rows=199413 width=32) (actual time=95.753..95.754 rows=199730 loops=1)
                                                        Buckets: 2048  Batches: 128  Memory Usage: 108kB
                                                        Buffers: shared hit=4124, temp written=1190
                                                        I/O Timings: temp write=8.023
                                                        ->  Seq Scan on rfv_text v_2  (cost=0.00..11124.00 rows=199413 width=32) (actual time=0.016..57.547 rows=199730 loops=1)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid) AND (value = 'x'::text COLLATE "C"))
                                                              Rows Removed by Filter: 200270
                                                              Buffers: shared hit=4124
                                            ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (actual time=91.470..91.470 rows=199887 loops=1)
                                                  Buckets: 2048  Batches: 128  Memory Usage: 115kB
                                                  Buffers: shared hit=4124, temp written=1189
                                                  I/O Timings: temp write=8.361
                                                  ->  Seq Scan on rfv_bool v  (cost=0.00..10124.00 rows=199887 width=32) (actual time=0.013..50.285 rows=199887 loops=1)
                                                        Filter: (value AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                        Rows Removed by Filter: 200113
                                                        Buffers: shared hit=4124
                    ->  Subquery Scan on "*SELECT* 2"  (cost=60987.64..64423.13 rows=1 width=0) (never executed)
                          ->  Limit  (cost=60987.64..64423.12 rows=1 width=4) (never executed)
                                ->  Result  (cost=60987.64..64423.12 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=60987.64..64423.12 rows=1 width=0) (never executed)
                                            Hash Cond: (v_3.record_id = v_5.record_id)
                                            ->  Hash Semi Join  (cost=46006.97..48076.44 rows=1 width=64) (never executed)
                                                  Hash Cond: (r_1.id = v_3.record_id)
                                                  ->  Hash Semi Join  (cost=32017.38..32717.85 rows=1 width=48) (never executed)
                                                        Hash Cond: (r_1.id = v_4.record_id)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        ->  Hash  (cost=30236.02..30236.02 rows=92065 width=32) (never executed)
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..30236.02 rows=92065 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                                    Heap Fetches: 0
                                                  ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (never executed)
                                                        ->  Seq Scan on rfv_bool v_3  (cost=0.00..10124.00 rows=199887 width=32) (never executed)
                                                              Filter: (value AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                            ->  Hash  (cost=11124.00..11124.00 rows=199413 width=32) (never executed)
                                                  ->  Seq Scan on rfv_text v_5  (cost=0.00..11124.00 rows=199413 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid) AND (value = 'x'::text COLLATE "C"))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=60987.64..64411.08 rows=1 width=0) (never executed)
                          ->  Limit  (cost=60987.64..64411.07 rows=1 width=4) (never executed)
                                ->  Result  (cost=60987.64..64411.07 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=60987.64..64411.07 rows=1 width=0) (never executed)
                                            Hash Cond: (v_6.record_id = v_8.record_id)
                                            ->  Hash Semi Join  (cost=46006.97..48064.39 rows=1 width=64) (never executed)
                                                  Hash Cond: (r_2.id = v_6.record_id)
                                                  ->  Hash Semi Join  (cost=32017.38..32705.80 rows=1 width=48) (never executed)
                                                        Hash Cond: (r_2.id = v_7.record_id)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                        ->  Hash  (cost=30236.02..30236.02 rows=92065 width=32) (never executed)
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..30236.02 rows=92065 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                                    Heap Fetches: 0
                                                  ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (never executed)
                                                        ->  Seq Scan on rfv_bool v_6  (cost=0.00..10124.00 rows=199887 width=32) (never executed)
                                                              Filter: (value AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                            ->  Hash  (cost=11124.00..11124.00 rows=199413 width=32) (never executed)
                                                  ->  Seq Scan on rfv_text v_8  (cost=0.00..11124.00 rows=199413 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid) AND (value = 'x'::text COLLATE "C"))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=61011.02..282859.93 rows=1 width=0) (never executed)
                          ->  Limit  (cost=61011.02..282859.92 rows=1 width=4) (never executed)
                                ->  Result  (cost=61011.02..282859.92 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=61011.02..282859.92 rows=1 width=0) (never executed)
                                            Hash Cond: (v_9.record_id = v_11.record_id)
                                            ->  Hash Semi Join  (cost=46030.34..266513.24 rows=1 width=80) (never executed)
                                                  Hash Cond: (v_9.record_id = v_10.record_id)
                                                  ->  Hash Semi Join  (cost=14013.52..233864.41 rows=1 width=64) (never executed)
                                                        Hash Cond: (r_3.id = v_9.record_id)
                                                        ->  Hash Semi Join  (cost=23.93..218505.82 rows=1 width=48) (never executed)
                                                              Hash Cond: (r_3.id = acl_share.record_id)
                                                              ->  Seq Scan on records r_3  (cost=0.00..218120.60 rows=137635 width=32) (never executed)
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                              ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (never executed)
                                                              ->  Seq Scan on rfv_bool v_9  (cost=0.00..10124.00 rows=199887 width=32) (never executed)
                                                                    Filter: (value AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                  ->  Hash  (cost=30236.02..30236.02 rows=92065 width=32) (never executed)
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_10  (cost=0.55..30236.02 rows=92065 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                              Heap Fetches: 0
                                            ->  Hash  (cost=11124.00..11124.00 rows=199413 width=32) (never executed)
                                                  ->  Seq Scan on rfv_text v_11  (cost=0.00..11124.00 rows=199413 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid) AND (value = 'x'::text COLLATE "C"))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 2.864 ms
Execution Time: 481.362 ms
```

### Cell parallel, unbudgeted count: a50, pk = 3 (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=27627.32..27627.33 rows=1 width=8) (actual time=120.183..153.884 rows=1 loops=1)
  Buffers: shared hit=9074
  ->  Limit  (cost=26203.75..27614.81 rows=1001 width=4) (actual time=113.057..153.631 rows=1001 loops=1)
        Buffers: shared hit=9074
        ->  Result  (cost=26203.75..27619.04 rows=1004 width=4) (actual time=113.055..153.120 rows=1001 loops=1)
              Buffers: shared hit=9074
              ->  Append  (cost=26203.75..27609.00 rows=1004 width=0) (actual time=113.053..152.853 rows=1001 loops=1)
                    Buffers: shared hit=9074
                    ->  Subquery Scan on "*SELECT* 1"  (cost=26203.75..27275.20 rows=1001 width=0) (actual time=113.051..152.401 rows=1001 loops=1)
                          Buffers: shared hit=9074
                          ->  Limit  (cost=26203.75..27265.19 rows=1001 width=4) (actual time=113.049..151.933 rows=1001 loops=1)
                                Buffers: shared hit=9074
                                ->  Gather  (cost=26203.75..32267.02 rows=5718 width=4) (actual time=113.047..151.734 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=9074
                                      ->  Result  (cost=25203.75..30695.22 rows=2382 width=4) (actual time=104.595..108.700 rows=592 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=9074
                                            ->  Parallel Hash Semi Join  (cost=25203.75..30695.22 rows=2382 width=0) (actual time=104.578..108.522 rows=592 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=9074
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5407.82 rows=20842 width=32) (actual time=0.401..2.152 rows=1212 loops=3)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1870
                                                  ->  Parallel Hash  (cost=24723.69..24723.69 rows=38360 width=32) (actual time=103.173..103.177 rows=51950 loops=3)
                                                        Buckets: 262144 (originally 131072)  Batches: 1 (originally 1)  Memory Usage: 0kB
                                                        Buffers: shared hit=7146
                                                        ->  Parallel Bitmap Heap Scan on rfv_num v  (cost=6978.38..24723.69 rows=38360 width=32) (actual time=25.665..59.746 rows=51950 loops=3)
                                                              Recheck Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                              Heap Blocks: exact=1655
                                                              Buffers: shared hit=7146
                                                              ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..6955.36 rows=92065 width=0) (actual time=30.969..30.970 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision))
                                                                    Buffers: shared hit=2600
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..160.05 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..160.04 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..160.04 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..160.04 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..134.83 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..134.82 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..134.82 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..134.82 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.779 ms
Execution Time: 154.175 ms
```

### Cell parallel, unbudgeted count: a6, three broad fields (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10835.97..10835.98 rows=1 width=8) (actual time=615.444..661.028 rows=1 loops=1)
  Buffers: shared hit=44864
  ->  Limit  (cost=8666.49..10833.77 rows=176 width=4) (actual time=443.067..660.537 rows=1001 loops=1)
        Buffers: shared hit=44864
        ->  Result  (cost=8666.49..10833.77 rows=176 width=4) (actual time=443.063..660.105 rows=1001 loops=1)
              Buffers: shared hit=44864
              ->  Append  (cost=8666.49..10832.01 rows=176 width=0) (actual time=443.062..659.650 rows=1001 loops=1)
                    Buffers: shared hit=44864
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10454.32 rows=173 width=0) (actual time=443.059..659.260 rows=1001 loops=1)
                          Buffers: shared hit=44864
                          ->  Limit  (cost=8666.49..10452.59 rows=173 width=4) (actual time=443.057..658.688 rows=1001 loops=1)
                                Buffers: shared hit=44864
                                ->  Gather  (cost=8666.49..10452.59 rows=173 width=4) (actual time=443.056..647.633 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=44864
                                      ->  Result  (cost=7666.49..9435.29 rows=102 width=4) (actual time=424.179..589.119 rows=588 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=44864
                                            ->  Nested Loop Semi Join  (cost=7666.49..9435.29 rows=102 width=0) (actual time=424.155..588.554 rows=588 loops=2)
                                                  Buffers: shared hit=44864
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8915.33 rows=444 width=64) (actual time=424.004..521.847 rows=1447 loops=2)
                                                        Buffers: shared hit=33287
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8372.64 rows=889 width=48) (actual time=423.871..463.464 rows=2900 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=10082
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.27 rows=3582 width=32) (actual time=0.476..20.318 rows=5743 loops=2)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5929
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=414.421..414.424 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=2.907..161.362 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.019..0.019 rows=0 loops=5801)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v.record_id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=23205
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.43..1.16 rows=1 width=32) (actual time=0.041..0.041 rows=0 loops=2894)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v.record_id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=11577
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..192.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..192.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..192.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..192.47 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..191.84 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..190.24 rows=2 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=16 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_4.record_id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..149.31 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..149.30 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..149.30 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..149.30 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..148.67 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..147.87 rows=1 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_7.record_id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_8.record_id) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..35.02 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..35.01 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..35.01 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..35.01 rows=1 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.83 rows=1 width=80) (never executed)
                                                  Join Filter: (r_3.id = v_9.record_id)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.21 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5fc05368-f599-4500-be05-7cbc393a2612}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '6b0205ca-76fd-44bf-aaa2-b73125ed1dfe'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '37e1af75-adb4-4047-b604-7daf6029ef69'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                  ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_11.record_id) AND (field_id = 'be960888-f13c-4e77-9caf-cb6de4f70174'::uuid))
                                                        Filter: value
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.43..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = v_9.record_id) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 35.632 ms
Execution Time: 661.749 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 (parallel_setup_cost = 0, parallel_tuple_cost = 0)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9084.35..9084.36 rows=1 width=8) (actual time=4445.012..4445.032 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9071.84 rows=1001 width=4) (actual time=11.330..4433.295 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9162.46 rows=1011 width=4) (actual time=11.327..4433.098 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9152.35 rows=1011 width=0) (actual time=11.325..4432.921 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8820.93 rows=1001 width=0) (actual time=11.323..4432.523 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8810.92 rows=1001 width=4) (actual time=11.321..4432.314 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169537.05 rows=19263 width=4) (actual time=11.320..4432.183 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169537.05 rows=19263 width=0) (actual time=11.301..4431.937 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.084..463.289 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..158.72 rows=5 width=0) (actual time=0.090..0.093 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..158.67 rows=5 width=4) (actual time=0.088..0.090 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..158.67 rows=5 width=4) (actual time=0.083..0.085 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..158.67 rows=5 width=0) (actual time=0.055..0.056 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.053..0.054 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..133.73 rows=4 width=0) (actual time=0.043..0.046 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..133.69 rows=4 width=4) (actual time=0.042..0.044 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..133.69 rows=4 width=4) (actual time=0.041..0.042 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..133.69 rows=4 width=0) (actual time=0.034..0.035 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.033..0.034 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.93 rows=1 width=0) (actual time=0.059..0.065 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.92 rows=1 width=4) (actual time=0.057..0.062 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.92 rows=1 width=4) (actual time=0.056..0.061 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.92 rows=1 width=0) (actual time=0.049..0.054 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.049..0.052 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.048..0.050 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.042..0.044 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.037..0.039 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.51 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', parallel_setup_cost = '0', parallel_tuple_cost = '0'
Planning Time: 12.467 ms
Execution Time: 4445.346 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 (max_parallel_workers_per_gather = 1, parallel_setup_cost = 0, parallel_tuple_cost = 0)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9084.35..9084.36 rows=1 width=8) (actual time=2655.721..2655.736 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9071.84 rows=1001 width=4) (actual time=6.219..2655.338 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9162.46 rows=1011 width=4) (actual time=6.218..2655.168 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9152.35 rows=1011 width=0) (actual time=6.217..2655.018 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8820.93 rows=1001 width=0) (actual time=6.215..2654.699 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8810.92 rows=1001 width=4) (actual time=6.214..2654.539 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169537.05 rows=19263 width=4) (actual time=6.212..2654.432 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169537.05 rows=19263 width=0) (actual time=6.200..2654.249 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5699.60 rows=50020 width=32) (actual time=0.059..194.842 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..158.72 rows=5 width=0) (actual time=0.054..0.056 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..158.67 rows=5 width=4) (actual time=0.053..0.054 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..158.67 rows=5 width=4) (actual time=0.049..0.050 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..158.67 rows=5 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.030..0.030 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..133.73 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..133.69 rows=4 width=4) (actual time=0.026..0.027 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..133.69 rows=4 width=4) (actual time=0.025..0.026 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..133.69 rows=4 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.021..0.021 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.93 rows=1 width=0) (actual time=0.036..0.040 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.92 rows=1 width=4) (actual time=0.035..0.039 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.92 rows=1 width=4) (actual time=0.034..0.037 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.92 rows=1 width=0) (actual time=0.031..0.033 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.030..0.032 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.029..0.031 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.025..0.027 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.51 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (record_id = r_3.id) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '1', parallel_setup_cost = '0', parallel_tuple_cost = '0'
Planning Time: 2.311 ms
Execution Time: 2655.943 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 (enable_nestloop = off)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=373643.93..373643.94 rows=1 width=8) (actual time=907.100..960.205 rows=1 loops=1)
  Buffers: shared hit=143612, temp read=14070 written=14200
  I/O Timings: temp read=44.478 write=199.019
  ->  Limit  (cost=31732.70..373631.41 rows=1001 width=4) (actual time=479.357..960.119 rows=100 loops=1)
        Buffers: shared hit=143612, temp read=14070 written=14200
        I/O Timings: temp read=44.478 write=199.019
        ->  Result  (cost=31732.70..377046.99 rows=1011 width=4) (actual time=479.355..960.082 rows=100 loops=1)
              Buffers: shared hit=143612, temp read=14070 written=14200
              I/O Timings: temp read=44.478 write=199.019
              ->  Append  (cost=31732.70..377036.88 rows=1011 width=0) (actual time=479.354..960.010 rows=100 loops=1)
                    Buffers: shared hit=143612, temp read=14070 written=14200
                    I/O Timings: temp read=44.478 write=199.019
                    ->  Subquery Scan on "*SELECT* 1"  (cost=31732.70..32196.35 rows=1001 width=0) (actual time=479.351..581.608 rows=100 loops=1)
                          Buffers: shared hit=126449, temp read=7679 written=7768
                          I/O Timings: temp read=28.157 write=129.255
                          ->  Limit  (cost=31732.70..32186.34 rows=1001 width=4) (actual time=479.349..581.545 rows=100 loops=1)
                                Buffers: shared hit=126449, temp read=7679 written=7768
                                I/O Timings: temp read=28.157 write=129.255
                                ->  Gather  (cost=31732.70..40462.30 rows=19263 width=4) (actual time=479.347..581.507 rows=100 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=126449, temp read=7679 written=7768
                                      I/O Timings: temp read=28.157 write=129.255
                                      ->  Result  (cost=30732.70..37536.00 rows=8026 width=4) (actual time=486.201..541.215 rows=33 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=126449, temp read=7679 written=7768
                                            I/O Timings: temp read=28.157 write=129.255
                                            ->  Parallel Hash Semi Join  (cost=30732.70..37536.00 rows=8026 width=0) (actual time=486.184..541.167 rows=33 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=126449, temp read=7679 written=7768
                                                  I/O Timings: temp read=28.157 write=129.255
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5407.82 rows=20842 width=32) (actual time=0.377..77.447 rows=66793 loops=3)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109317
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=139734 width=32) (actual time=357.662..357.663 rows=234155 loops=3)
                                                        Buckets: 131072 (originally 131072)  Batches: 8 (originally 4)  Memory Usage: 6560kB
                                                        Buffers: shared hit=17074, temp read=2478 written=6440
                                                        I/O Timings: temp read=8.314 write=105.619
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..28029.47 rows=139734 width=32) (actual time=67.388..179.052 rows=234155 loops=3)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                              Rows Removed by Filter: 266667
                                                              Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=49852.69..52216.28 rows=5 width=0) (actual time=0.076..0.084 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49852.69..52216.23 rows=5 width=4) (actual time=0.074..0.080 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49852.69..52216.23 rows=5 width=4) (actual time=0.070..0.076 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49852.69..52216.23 rows=5 width=0) (actual time=0.051..0.056 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.049..0.050 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=49852.69..52204.21 rows=4 width=0) (actual time=0.028..0.032 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49852.69..52204.17 rows=4 width=4) (actual time=0.026..0.029 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49852.69..52204.17 rows=4 width=4) (actual time=0.025..0.028 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49852.69..52204.17 rows=4 width=0) (actual time=0.022..0.024 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.021..0.021 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                        Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=31756.08..240414.98 rows=1 width=0) (actual time=355.654..378.231 rows=0 loops=1)
                          Buffers: shared hit=17155, temp read=6391 written=6432
                          I/O Timings: temp read=16.321 write=69.764
                          ->  Limit  (cost=31756.08..240414.97 rows=1 width=4) (actual time=355.652..378.228 rows=0 loops=1)
                                Buffers: shared hit=17155, temp read=6391 written=6432
                                I/O Timings: temp read=16.321 write=69.764
                                ->  Gather  (cost=31756.08..240414.97 rows=1 width=4) (actual time=355.650..378.224 rows=0 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=17155, temp read=6391 written=6432
                                      I/O Timings: temp read=16.321 write=69.764
                                      ->  Result  (cost=30756.08..239414.87 rows=1 width=4) (actual time=345.404..345.699 rows=0 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=17155, temp read=6391 written=6432
                                            I/O Timings: temp read=16.321 write=69.764
                                            ->  Parallel Hash Semi Join  (cost=30756.08..239414.87 rows=1 width=0) (actual time=345.387..345.681 rows=0 loops=3)
                                                  Hash Cond: (r_3.id = v_3.record_id)
                                                  Buffers: shared hit=17155, temp read=6391 written=6432
                                                  I/O Timings: temp read=16.321 write=69.764
                                                  ->  Hash Semi Join  (cost=23.93..207724.72 rows=1 width=48) (actual time=0.242..0.247 rows=0 loops=3)
                                                        Hash Cond: (r_3.id = acl_share.record_id)
                                                        Buffers: shared hit=23
                                                        ->  Parallel Seq Scan on records r_3  (cost=0.00..207550.25 rows=57348 width=32) (actual time=0.053..0.054 rows=1 loops=3)
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=3
                                                        ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.118..0.121 rows=0 loops=3)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                              Buffers: shared hit=20
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.117..0.119 rows=0 loops=3)
                                                                    Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                    Buffers: shared hit=20
                                                                    ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.086..0.087 rows=0 loops=3)
                                                                          Buffers: shared hit=20
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.062..0.062 rows=0 loops=3)
                                                                                Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                                Buffers: shared hit=8
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.022..0.022 rows=0 loops=3)
                                                                                Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                                Buffers: shared hit=12
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=139734 width=32) (actual time=300.995..300.996 rows=234155 loops=3)
                                                        Buckets: 131072 (originally 131072)  Batches: 8 (originally 4)  Memory Usage: 6592kB
                                                        Buffers: shared hit=17074, temp read=2476 written=6432
                                                        I/O Timings: temp read=6.656 write=69.764
                                                        ->  Parallel Seq Scan on rfv_num v_3  (cost=0.00..28029.47 rows=139734 width=32) (actual time=63.094..156.577 rows=234155 loops=3)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                              Rows Removed by Filter: 266667
                                                              Buffers: shared hit=17074
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off'
Planning Time: 2.921 ms
Execution Time: 960.492 ms
```

### Cell parallel, unbudgeted count: a50, pk < 5 (enable_nestloop = off)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=376519.87..376519.88 rows=1 width=8) (actual time=281.035..310.015 rows=1 loops=1)
  Buffers: shared hit=126446, temp read=1004 written=2792
  I/O Timings: temp read=3.338 write=32.489
  ->  Limit  (cost=32059.24..376507.36 rows=1001 width=4) (actual time=278.131..309.869 rows=1001 loops=1)
        Buffers: shared hit=126446, temp read=1004 written=2792
        I/O Timings: temp read=3.338 write=32.489
        ->  Result  (cost=32059.24..379948.40 rows=1011 width=4) (actual time=278.129..309.705 rows=1001 loops=1)
              Buffers: shared hit=126446, temp read=1004 written=2792
              I/O Timings: temp read=3.338 write=32.489
              ->  Append  (cost=32059.24..379938.29 rows=1011 width=0) (actual time=278.127..309.511 rows=1001 loops=1)
                    Buffers: shared hit=126446, temp read=1004 written=2792
                    I/O Timings: temp read=3.338 write=32.489
                    ->  Subquery Scan on "*SELECT* 1"  (cost=32059.24..32528.86 rows=1001 width=0) (actual time=278.126..309.336 rows=1001 loops=1)
                          Buffers: shared hit=126446, temp read=1004 written=2792
                          I/O Timings: temp read=3.338 write=32.489
                          ->  Limit  (cost=32059.24..32518.85 rows=1001 width=4) (actual time=278.123..309.167 rows=1001 loops=1)
                                Buffers: shared hit=126446, temp read=1004 written=2792
                                I/O Timings: temp read=3.338 write=32.489
                                ->  Gather  (cost=32059.24..40903.84 rows=19263 width=4) (actual time=278.122..309.036 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=126446, temp read=1004 written=2792
                                      I/O Timings: temp read=3.338 write=32.489
                                      ->  Result  (cost=31059.24..37977.54 rows=8026 width=4) (actual time=282.839..283.866 rows=334 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=126446, temp read=1004 written=2792
                                            I/O Timings: temp read=3.338 write=32.489
                                            ->  Parallel Hash Semi Join  (cost=31059.24..37977.54 rows=8026 width=0) (actual time=282.824..283.789 rows=334 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=126446, temp read=1004 written=2792
                                                  I/O Timings: temp read=3.338 write=32.489
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5407.82 rows=20842 width=32) (actual time=0.122..58.669 rows=66793 loops=3)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109314
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=156657 width=32) (actual time=187.957..187.958 rows=103957 loops=3)
                                                        Buckets: 131072  Batches: 4  Memory Usage: 0kB
                                                        Buffers: shared hit=17074, temp written=1504
                                                        I/O Timings: temp write=18.830
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..28029.47 rows=156657 width=32) (actual time=32.179..135.180 rows=103957 loops=3)
                                                              Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                              Rows Removed by Filter: 396864
                                                              Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=50638.38..53279.97 rows=5 width=0) (never executed)
                          ->  Limit  (cost=50638.38..53279.92 rows=5 width=4) (never executed)
                                ->  Result  (cost=50638.38..53279.92 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=50638.38..53279.92 rows=5 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=375976 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=50638.38..53267.89 rows=4 width=0) (never executed)
                          ->  Limit  (cost=50638.38..53267.85 rows=4 width=4) (never executed)
                                ->  Result  (cost=50638.38..53267.85 rows=4 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      ->  Hash Semi Join  (cost=50638.38..53267.85 rows=4 width=0) (never executed)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=375976 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=32082.62..240856.52 rows=1 width=0) (never executed)
                          ->  Limit  (cost=32082.62..240856.51 rows=1 width=4) (never executed)
                                ->  Gather  (cost=32082.62..240856.51 rows=1 width=4) (never executed)
                                      Workers Planned: 2
                                      Workers Launched: 0
                                      ->  Result  (cost=31082.62..239856.41 rows=1 width=4) (never executed)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            ->  Parallel Hash Semi Join  (cost=31082.62..239856.41 rows=1 width=0) (never executed)
                                                  Hash Cond: (r_3.id = v_3.record_id)
                                                  ->  Hash Semi Join  (cost=23.93..207724.72 rows=1 width=48) (never executed)
                                                        Hash Cond: (r_3.id = acl_share.record_id)
                                                        ->  Parallel Seq Scan on records r_3  (cost=0.00..207550.25 rows=57348 width=32) (never executed)
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                        ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                    Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                    ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=156657 width=32) (never executed)
                                                        ->  Parallel Seq Scan on rfv_num v_3  (cost=0.00..28029.47 rows=156657 width=32) (never executed)
                                                              Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off'
Planning Time: 2.245 ms
Execution Time: 310.198 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 and pk < 5 (enable_nestloop = off)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=534033.83..534033.84 rows=1 width=8) (actual time=3797.770..3831.780 rows=1 loops=1)
  Buffers: shared hit=515198, temp read=7881 written=10920
  I/O Timings: temp read=37.795 write=213.291
  ->  Limit  (cost=4121.91..534021.32 rows=1001 width=4) (actual time=0.153..3831.751 rows=100 loops=1)
        Buffers: shared hit=515198, temp read=7881 written=10920
        I/O Timings: temp read=37.795 write=213.291
        ->  Result  (cost=4121.91..539315.02 rows=1011 width=4) (actual time=0.151..3831.730 rows=100 loops=1)
              Buffers: shared hit=515198, temp read=7881 written=10920
              I/O Timings: temp read=37.795 write=213.291
              ->  Append  (cost=4121.91..539304.91 rows=1011 width=0) (actual time=0.150..3831.704 rows=100 loops=1)
                    Buffers: shared hit=515198, temp read=7881 written=10920
                    I/O Timings: temp read=37.795 write=213.291
                    ->  Subquery Scan on "*SELECT* 1"  (cost=4121.91..55910.98 rows=1001 width=0) (actual time=0.149..1634.901 rows=100 loops=1)
                          Buffers: shared hit=446812
                          ->  Limit  (cost=4121.91..55900.97 rows=1001 width=4) (actual time=0.148..1634.877 rows=100 loops=1)
                                Buffers: shared hit=446812
                                ->  Result  (cost=4121.91..1000545.46 rows=19263 width=4) (actual time=0.148..1634.857 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=446812
                                      ->  Merge Semi Join  (cost=4121.91..1000545.46 rows=19263 width=0) (actual time=0.135..1634.820 rows=100 loops=1)
                                            Merge Cond: (v.record_id = v_1.record_id)
                                            Buffers: shared hit=446812
                                            ->  Merge Semi Join  (cost=4121.34..947271.11 rows=19263 width=48) (actual time=0.097..1229.508 rows=200376 loops=1)
                                                  Merge Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=427116
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..824303.09 rows=50020 width=32) (actual time=0.073..906.001 rows=200376 loops=1)
                                                        Index Cond: (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                        Rows Removed by Filter: 199621
                                                        Buffers: shared hit=404376
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..125790.02 rows=375976 width=32) (actual time=0.022..206.849 rows=311868 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Filter: (value < '5'::double precision)
                                                        Rows Removed by Filter: 88129
                                                        Buffers: shared hit=22740
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..52179.49 rows=335361 width=32) (actual time=0.029..289.057 rows=702464 loops=1)
                                                  Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                  Heap Fetches: 702464
                                                  Buffers: shared hit=19696
                    ->  Subquery Scan on "*SELECT* 2"  (cost=100490.51..105427.15 rows=5 width=0) (actual time=613.951..613.960 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=1486
                          I/O Timings: temp write=41.811
                          ->  Limit  (cost=100490.51..105427.10 rows=5 width=4) (actual time=613.949..613.957 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=1486
                                I/O Timings: temp write=41.811
                                ->  Result  (cost=100490.51..105427.10 rows=5 width=4) (actual time=613.946..613.952 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=17078, temp written=1486
                                      I/O Timings: temp write=41.811
                                      ->  Hash Semi Join  (cost=100490.51..105427.10 rows=5 width=0) (actual time=613.926..613.932 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=17078, temp written=1486
                                            I/O Timings: temp write=41.811
                                            ->  Hash Semi Join  (cost=49852.68..52216.22 rows=5 width=48) (actual time=0.098..0.101 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_3.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.55..68.98 rows=12 width=32) (actual time=0.095..0.095 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=608.230..608.231 rows=311871 loops=1)
                                                  Buckets: 131072  Batches: 4  Memory Usage: 5898kB
                                                  Buffers: shared hit=17074, temp written=1483
                                                  I/O Timings: temp write=41.655
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=375976 width=32) (actual time=65.060..409.114 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 3"  (cost=100490.51..105415.07 rows=4 width=0) (actual time=627.312..627.320 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=1486
                          I/O Timings: temp write=26.354
                          ->  Limit  (cost=100490.51..105415.03 rows=4 width=4) (actual time=627.307..627.313 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=1486
                                I/O Timings: temp write=26.354
                                ->  Result  (cost=100490.51..105415.03 rows=4 width=4) (actual time=627.303..627.308 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                      Buffers: shared hit=17078, temp written=1486
                                      I/O Timings: temp write=26.354
                                      ->  Hash Semi Join  (cost=100490.51..105415.03 rows=4 width=0) (actual time=627.277..627.281 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=17078, temp written=1486
                                            I/O Timings: temp write=26.354
                                            ->  Hash Semi Join  (cost=49852.68..52204.15 rows=4 width=48) (actual time=0.088..0.090 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_5.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..56.93 rows=10 width=32) (actual time=0.085..0.086 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335361 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_5  (cost=0.00..43367.12 rows=335361 width=32) (never executed)
                                                              Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=375976 width=32) (actual time=620.408..620.409 rows=311871 loops=1)
                                                  Buckets: 131072  Batches: 4  Memory Usage: 5898kB
                                                  Buffers: shared hit=17074, temp written=1483
                                                  I/O Timings: temp write=26.218
                                                  ->  Seq Scan on rfv_num v_4  (cost=0.00..43367.12 rows=375976 width=32) (actual time=83.908..467.204 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 4"  (cost=62814.76..272546.67 rows=1 width=0) (actual time=921.512..955.488 rows=0 loops=1)
                          Buffers: shared hit=34230, temp read=7881 written=7948
                          I/O Timings: temp read=37.795 write=145.126
                          ->  Limit  (cost=62814.76..272546.66 rows=1 width=4) (actual time=921.506..955.481 rows=0 loops=1)
                                Buffers: shared hit=34230, temp read=7881 written=7948
                                I/O Timings: temp read=37.795 write=145.126
                                ->  Gather  (cost=62814.76..272546.66 rows=1 width=4) (actual time=921.501..955.474 rows=0 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=34230, temp read=7881 written=7948
                                      I/O Timings: temp read=37.795 write=145.126
                                      ->  Result  (cost=61814.76..271546.56 rows=1 width=4) (actual time=907.292..907.307 rows=0 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid)
                                            Buffers: shared hit=34230, temp read=7881 written=7948
                                            I/O Timings: temp read=37.795 write=145.126
                                            ->  Parallel Hash Semi Join  (cost=61814.76..271546.56 rows=1 width=0) (actual time=907.274..907.289 rows=0 loops=3)
                                                  Hash Cond: (r_3.id = v_6.record_id)
                                                  Buffers: shared hit=34230, temp read=7881 written=7948
                                                  I/O Timings: temp read=37.795 write=145.126
                                                  ->  Parallel Hash Semi Join  (cost=30756.07..239414.86 rows=1 width=64) (actual time=543.740..543.752 rows=0 loops=3)
                                                        Hash Cond: (r_3.id = v_7.record_id)
                                                        Buffers: shared hit=17098, temp read=6391 written=6444
                                                        I/O Timings: temp read=32.010 write=100.483
                                                        ->  Hash Semi Join  (cost=23.93..207724.72 rows=1 width=48) (actual time=0.188..0.195 rows=0 loops=3)
                                                              Hash Cond: (r_3.id = acl_share.record_id)
                                                              Buffers: shared hit=24
                                                              ->  Parallel Seq Scan on records r_3  (cost=0.00..207550.25 rows=57348 width=32) (actual time=0.067..0.068 rows=1 loops=3)
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (entity_definition_id = '01a2017c-6606-7ca0-bacd-8a6ee7c9e8bc'::uuid))
                                                                    Rows Removed by Filter: 2
                                                                    Buffers: shared hit=4
                                                              ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.092..0.097 rows=0 loops=3)
                                                                    Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                                    Buffers: shared hit=20
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.091..0.094 rows=0 loops=3)
                                                                          Recheck Cond: (((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[]))))
                                                                          Buffers: shared hit=20
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.088..0.090 rows=0 loops=3)
                                                                                Buffers: shared hit=20
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.058..0.058 rows=0 loops=3)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = '9bac2af2-f42d-4677-81a1-4246cd19af50'::uuid))
                                                                                      Buffers: shared hit=8
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.027..0.028 rows=0 loops=3)
                                                                                      Index Cond: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,6e6ea293-aa9e-4c9d-a67e-c33d96b86693}'::uuid[])))
                                                                                      Buffers: shared hit=12
                                                        ->  Parallel Hash  (cost=28029.47..28029.47 rows=139734 width=32) (actual time=466.428..466.429 rows=234155 loops=3)
                                                              Buckets: 131072 (originally 131072)  Batches: 8 (originally 4)  Memory Usage: 6592kB
                                                              Buffers: shared hit=17074, temp read=2476 written=6444
                                                              I/O Timings: temp read=7.413 write=100.483
                                                              ->  Parallel Seq Scan on rfv_num v_7  (cost=0.00..28029.47 rows=139734 width=32) (actual time=105.704..248.720 rows=234155 loops=3)
                                                                    Filter: ((tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = '3a2969a4-db3a-4c1f-b9fa-184a6517365c'::uuid) AND (value = '1'::double precision))
                                                                    Rows Removed by Filter: 266667
                                                                    Buffers: shared hit=17074
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=156657 width=32) (actual time=342.189..342.191 rows=103957 loops=3)
                                                        Buckets: 131072  Batches: 4  Memory Usage: 5952kB
                                                        Buffers: shared hit=17074, temp written=1504
                                                        I/O Timings: temp write=44.642
                                                        ->  Parallel Seq Scan on rfv_num v_6  (cost=0.00..28029.47 rows=156657 width=32) (actual time=49.512..240.603 rows=103957 loops=3)
                                                              Filter: ((value < '5'::double precision) AND (tenant_id = '6ccb2b3e-5005-4719-b202-e53184974033'::uuid) AND (field_id = 'fb97e835-10f5-4f68-b32d-36acaa724cb6'::uuid))
                                                              Rows Removed by Filter: 396864
                                                              Buffers: shared hit=17074
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off'
Planning Time: 4.264 ms
Execution Time: 3832.241 ms
```
