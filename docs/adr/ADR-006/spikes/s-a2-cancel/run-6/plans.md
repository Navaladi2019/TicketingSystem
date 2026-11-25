# TICK-1048 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)

### Cell spill, unbudgeted count: a50, pk = 3 (max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=18889.23..18889.24 rows=1 width=8) (actual time=201.157..201.167 rows=1 loops=1)
  Buffers: shared hit=109377, temp read=3109 written=4402
  I/O Timings: temp read=6.773 write=10.631
  ->  Limit  (cost=13221.82..18876.72 rows=1001 width=4) (actual time=199.243..201.060 rows=1001 loops=1)
        Buffers: shared hit=109377, temp read=3109 written=4402
        I/O Timings: temp read=6.773 write=10.631
        ->  Result  (cost=13221.82..18893.67 rows=1004 width=4) (actual time=199.242..200.933 rows=1001 loops=1)
              Buffers: shared hit=109377, temp read=3109 written=4402
              I/O Timings: temp read=6.773 write=10.631
              ->  Append  (cost=13221.82..18883.63 rows=1004 width=0) (actual time=199.240..200.784 rows=1001 loops=1)
                    Buffers: shared hit=109377, temp read=3109 written=4402
                    I/O Timings: temp read=6.773 write=10.631
                    ->  Subquery Scan on "*SELECT* 1"  (cost=13221.82..18577.76 rows=1001 width=0) (actual time=199.239..200.682 rows=1001 loops=1)
                          Buffers: shared hit=109377, temp read=3109 written=4402
                          I/O Timings: temp read=6.773 write=10.631
                          ->  Limit  (cost=13221.82..18567.75 rows=1001 width=4) (actual time=199.238..200.566 rows=1001 loops=1)
                                Buffers: shared hit=109377, temp read=3109 written=4402
                                I/O Timings: temp read=6.773 write=10.631
                                ->  Result  (cost=13221.82..43978.26 rows=5759 width=4) (actual time=199.237..200.476 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=109377, temp read=3109 written=4402
                                      I/O Timings: temp read=6.773 write=10.631
                                      ->  Merge Semi Join  (cost=13221.82..43978.26 rows=5759 width=0) (actual time=199.227..200.342 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=109377, temp read=3109 written=4402
                                            I/O Timings: temp read=6.773 write=10.631
                                            ->  Sort  (cost=13220.01..13344.81 rows=49923 width=32) (actual time=199.163..199.414 rows=2026 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: external merge  Disk: 8296kB
                                                  Buffers: shared hit=109306, temp read=3109 written=4402
                                                  I/O Timings: temp read=6.773 write=10.631
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.038..59.840 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109306
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..30222.26 rows=92012 width=32) (actual time=0.054..0.428 rows=1126 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 1126
                                                  Buffers: shared hit=71
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..144.57 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..144.56 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..144.56 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..144.56 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..122.39 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..122.38 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..122.38 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..122.38 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r_3.id) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.558 ms
Execution Time: 203.002 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 (max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9056.97..9056.98 rows=1 width=8) (actual time=833.883..833.894 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9044.46 rows=1001 width=4) (actual time=3.746..833.632 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9134.80 rows=1011 width=4) (actual time=3.745..833.508 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9124.69 rows=1011 width=0) (actual time=3.744..833.389 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8821.51 rows=1001 width=0) (actual time=3.743..833.182 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8811.50 rows=1001 width=4) (actual time=3.743..833.038 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169249.06 rows=19229 width=4) (actual time=3.742..832.960 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169249.06 rows=19229 width=0) (actual time=3.736..832.734 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.023..53.117 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r.id) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..143.06 rows=5 width=0) (actual time=0.039..0.041 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..143.01 rows=5 width=4) (actual time=0.038..0.040 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..143.01 rows=5 width=4) (actual time=0.035..0.036 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..143.01 rows=5 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.020..0.020 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..121.14 rows=4 width=0) (actual time=0.015..0.017 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..121.10 rows=4 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..121.10 rows=4 width=4) (actual time=0.014..0.015 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..121.10 rows=4 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.011..0.011 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.92 rows=1 width=0) (actual time=0.027..0.030 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.91 rows=1 width=4) (actual time=0.026..0.029 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.91 rows=1 width=4) (actual time=0.026..0.028 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.91 rows=1 width=0) (actual time=0.024..0.026 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.023..0.025 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.023..0.024 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.019..0.020 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.018..0.018 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.50 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r_3.id) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.380 ms
Execution Time: 833.992 ms
```

### Cell spill, unbudgeted count: a50, pk = 3 (enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=329626.64..329626.65 rows=1 width=8) (actual time=160.200..160.210 rows=1 loops=1)
  Buffers: shared hit=109377, temp read=3109 written=4402
  I/O Timings: temp read=4.326 write=7.918
  ->  Limit  (cost=13221.82..329614.13 rows=1001 width=4) (actual time=158.531..160.118 rows=1001 loops=1)
        Buffers: shared hit=109377, temp read=3109 written=4402
        I/O Timings: temp read=4.326 write=7.918
        ->  Result  (cost=13221.82..330562.36 rows=1004 width=4) (actual time=158.530..160.021 rows=1001 loops=1)
              Buffers: shared hit=109377, temp read=3109 written=4402
              I/O Timings: temp read=4.326 write=7.918
              ->  Append  (cost=13221.82..330552.32 rows=1004 width=0) (actual time=158.529..159.890 rows=1001 loops=1)
                    Buffers: shared hit=109377, temp read=3109 written=4402
                    I/O Timings: temp read=4.326 write=7.918
                    ->  Subquery Scan on "*SELECT* 1"  (cost=13221.82..18577.76 rows=1001 width=0) (actual time=158.527..159.800 rows=1001 loops=1)
                          Buffers: shared hit=109377, temp read=3109 written=4402
                          I/O Timings: temp read=4.326 write=7.918
                          ->  Limit  (cost=13221.82..18567.75 rows=1001 width=4) (actual time=158.526..159.703 rows=1001 loops=1)
                                Buffers: shared hit=109377, temp read=3109 written=4402
                                I/O Timings: temp read=4.326 write=7.918
                                ->  Result  (cost=13221.82..43978.26 rows=5759 width=4) (actual time=158.525..159.625 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=109377, temp read=3109 written=4402
                                      I/O Timings: temp read=4.326 write=7.918
                                      ->  Merge Semi Join  (cost=13221.82..43978.26 rows=5759 width=0) (actual time=158.519..159.512 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=109377, temp read=3109 written=4402
                                            I/O Timings: temp read=4.326 write=7.918
                                            ->  Sort  (cost=13220.01..13344.81 rows=49923 width=32) (actual time=158.473..158.695 rows=2026 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: external merge  Disk: 8296kB
                                                  Buffers: shared hit=109306, temp read=3109 written=4402
                                                  I/O Timings: temp read=4.326 write=7.918
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.029..46.589 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109306
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..30222.26 rows=92012 width=32) (actual time=0.038..0.360 rows=1126 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 1126
                                                  Buffers: shared hit=71
                    ->  Subquery Scan on "*SELECT* 2"  (cost=54.28..30506.09 rows=1 width=0) (never executed)
                          ->  Limit  (cost=54.28..30506.08 rows=1 width=4) (never executed)
                                ->  Result  (cost=54.28..30506.08 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Merge Semi Join  (cost=54.28..30506.08 rows=1 width=0) (never executed)
                                            Merge Cond: (r_1.id = v_1.record_id)
                                            ->  Sort  (cost=53.71..53.74 rows=12 width=32) (never executed)
                                                  Sort Key: r_1.id
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..30222.26 rows=92012 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=45.22..30497.03 rows=1 width=0) (never executed)
                          ->  Limit  (cost=45.22..30497.02 rows=1 width=4) (never executed)
                                ->  Result  (cost=45.22..30497.02 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Merge Semi Join  (cost=45.22..30497.02 rows=1 width=0) (never executed)
                                            Merge Cond: (r_2.id = v_2.record_id)
                                            ->  Sort  (cost=44.66..44.68 rows=10 width=32) (never executed)
                                                  Sort Key: r_2.id
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..30222.26 rows=92012 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=32025.35..250966.42 rows=1 width=0) (never executed)
                          ->  Limit  (cost=32025.35..250966.41 rows=1 width=4) (never executed)
                                ->  Result  (cost=32025.35..250966.41 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=32025.35..250966.41 rows=1 width=0) (never executed)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            ->  Hash Semi Join  (cost=23.93..218333.98 rows=1 width=48) (never executed)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  ->  Seq Scan on records r_3  (cost=0.00..217947.19 rows=138232 width=32) (never executed)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Hash  (cost=30222.26..30222.26 rows=92012 width=32) (never executed)
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.55..30222.26 rows=92012 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                        Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.193 ms
Execution Time: 161.456 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 (enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=409319.50..409319.51 rows=1 width=8) (actual time=684.016..684.024 rows=1 loops=1)
  Buffers: shared hit=424092
  ->  Limit  (cost=3161.27..409306.99 rows=1001 width=4) (actual time=0.057..684.011 rows=100 loops=1)
        Buffers: shared hit=424092
        ->  Result  (cost=3161.27..413364.39 rows=1011 width=4) (actual time=0.056..684.000 rows=100 loops=1)
              Buffers: shared hit=424092
              ->  Append  (cost=3161.27..413354.28 rows=1011 width=0) (actual time=0.055..683.987 rows=100 loops=1)
                    Buffers: shared hit=424092
                    ->  Subquery Scan on "*SELECT* 1"  (cost=3161.27..38464.19 rows=1001 width=0) (actual time=0.055..683.881 rows=100 loops=1)
                          Buffers: shared hit=424072
                          ->  Limit  (cost=3161.27..38454.18 rows=1001 width=4) (actual time=0.054..683.869 rows=100 loops=1)
                                Buffers: shared hit=424072
                                ->  Result  (cost=3161.27..681130.67 rows=19229 width=4) (actual time=0.054..683.860 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=424072
                                      ->  Merge Semi Join  (cost=3161.27..681130.67 rows=19229 width=0) (actual time=0.049..683.844 rows=100 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=424072
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..630958.15 rows=49923 width=32) (actual time=0.026..476.323 rows=200376 loops=1)
                                                  Index Cond: (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                  Rows Removed by Filter: 199621
                                                  Buffers: shared hit=404376
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..52159.24 rows=335503 width=32) (actual time=0.018..144.022 rows=702464 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                  Heap Fetches: 702464
                                                  Buffers: shared hit=19696
                    ->  Subquery Scan on "*SELECT* 2"  (cost=49855.47..52204.58 rows=5 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49855.47..52204.53 rows=5 width=4) (actual time=0.029..0.030 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49855.47..52204.53 rows=5 width=4) (actual time=0.028..0.029 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49855.47..52204.53 rows=5 width=0) (actual time=0.017..0.018 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.016..0.016 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=49855.47..52195.54 rows=4 width=0) (actual time=0.009..0.011 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49855.47..52195.50 rows=4 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49855.47..52195.50 rows=4 width=4) (actual time=0.008..0.010 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49855.47..52195.50 rows=4 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.006..0.006 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=49878.85..270484.91 rows=1 width=0) (actual time=0.048..0.050 rows=0 loops=1)
                          Buffers: shared hit=12
                          ->  Limit  (cost=49878.85..270484.90 rows=1 width=4) (actual time=0.048..0.049 rows=0 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=49878.85..270484.90 rows=1 width=4) (actual time=0.047..0.049 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=12
                                      ->  Hash Semi Join  (cost=49878.85..270484.90 rows=1 width=0) (actual time=0.046..0.047 rows=0 loops=1)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            Buffers: shared hit=12
                                            ->  Hash Semi Join  (cost=23.93..218333.98 rows=1 width=48) (actual time=0.045..0.046 rows=0 loops=1)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  Buffers: shared hit=12
                                                  ->  Seq Scan on records r_3  (cost=0.00..217947.19 rows=138232 width=32) (actual time=0.027..0.027 rows=1 loops=1)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=6
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.013..0.014 rows=0 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.005..0.005 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                          Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.185 ms
Execution Time: 684.111 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=421541.58..421541.59 rows=1 width=8) (actual time=800.098..800.108 rows=1 loops=1)
  Buffers: shared hit=126400, temp read=7920 written=7920
  I/O Timings: temp read=14.153 write=80.900
  ->  Limit  (cost=49855.47..421529.06 rows=1001 width=4) (actual time=637.800..800.029 rows=100 loops=1)
        Buffers: shared hit=126400, temp read=7920 written=7920
        I/O Timings: temp read=14.153 write=80.900
        ->  Result  (cost=49855.47..425242.09 rows=1011 width=4) (actual time=637.799..799.977 rows=100 loops=1)
              Buffers: shared hit=126400, temp read=7920 written=7920
              I/O Timings: temp read=14.153 write=80.900
              ->  Append  (cost=49855.47..425231.98 rows=1011 width=0) (actual time=637.798..799.926 rows=100 loops=1)
                    Buffers: shared hit=126400, temp read=7920 written=7920
                    I/O Timings: temp read=14.153 write=80.900
                    ->  Subquery Scan on "*SELECT* 1"  (cost=49855.47..50341.89 rows=1001 width=0) (actual time=637.797..799.718 rows=100 loops=1)
                          Buffers: shared hit=126380, temp read=7920 written=7920
                          I/O Timings: temp read=14.153 write=80.900
                          ->  Limit  (cost=49855.47..50331.88 rows=1001 width=4) (actual time=637.796..799.654 rows=100 loops=1)
                                Buffers: shared hit=126380, temp read=7920 written=7920
                                I/O Timings: temp read=14.153 write=80.900
                                ->  Result  (cost=49855.47..59007.20 rows=19229 width=4) (actual time=637.794..799.590 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=126380, temp read=7920 written=7920
                                      I/O Timings: temp read=14.153 write=80.900
                                      ->  Hash Semi Join  (cost=49855.47..59007.20 rows=19229 width=0) (actual time=637.787..799.519 rows=100 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=126380, temp read=7920 written=7920
                                            I/O Timings: temp read=14.153 write=80.900
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.025..69.788 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (actual time=497.409..497.410 rows=702464 loops=1)
                                                  Buckets: 2048 (originally 2048)  Batches: 512 (originally 256)  Memory Usage: 113kB
                                                  Buffers: shared hit=17074, temp written=4235
                                                  I/O Timings: temp write=55.227
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..43367.12 rows=335503 width=32) (actual time=76.538..259.130 rows=702464 loops=1)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                        Rows Removed by Filter: 800000
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=49855.47..52204.58 rows=5 width=0) (actual time=0.048..0.050 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49855.47..52204.53 rows=5 width=4) (actual time=0.047..0.049 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49855.47..52204.53 rows=5 width=4) (actual time=0.046..0.047 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49855.47..52204.53 rows=5 width=0) (actual time=0.033..0.034 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.032..0.032 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=49855.47..52195.54 rows=4 width=0) (actual time=0.017..0.018 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49855.47..52195.50 rows=4 width=4) (actual time=0.016..0.017 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49855.47..52195.50 rows=4 width=4) (actual time=0.015..0.016 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49855.47..52195.50 rows=4 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=49878.85..270484.91 rows=1 width=0) (actual time=0.088..0.090 rows=0 loops=1)
                          Buffers: shared hit=12
                          ->  Limit  (cost=49878.85..270484.90 rows=1 width=4) (actual time=0.087..0.089 rows=0 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=49878.85..270484.90 rows=1 width=4) (actual time=0.087..0.089 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=12
                                      ->  Hash Semi Join  (cost=49878.85..270484.90 rows=1 width=0) (actual time=0.085..0.086 rows=0 loops=1)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            Buffers: shared hit=12
                                            ->  Hash Semi Join  (cost=23.93..218333.98 rows=1 width=48) (actual time=0.084..0.086 rows=0 loops=1)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  Buffers: shared hit=12
                                                  ->  Seq Scan on records r_3  (cost=0.00..217947.19 rows=138232 width=32) (actual time=0.041..0.041 rows=1 loops=1)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                        Rows Removed by Filter: 33
                                                        Buffers: shared hit=6
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.018..0.019 rows=0 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                          Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.098 ms
Execution Time: 800.201 ms
```

### Cell spill, unbudgeted count: a50, pk < 5 (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=425233.54..425233.55 rows=1 width=8) (actual time=434.200..434.211 rows=1 loops=1)
  Buffers: shared hit=126380, temp read=10 written=2949
  I/O Timings: temp read=0.029 write=28.835
  ->  Limit  (cost=50588.55..425221.02 rows=1001 width=4) (actual time=313.543..434.010 rows=1001 loops=1)
        Buffers: shared hit=126380, temp read=10 written=2949
        I/O Timings: temp read=0.029 write=28.835
        ->  Result  (cost=50588.55..428963.61 rows=1011 width=4) (actual time=313.542..433.848 rows=1001 loops=1)
              Buffers: shared hit=126380, temp read=10 written=2949
              I/O Timings: temp read=0.029 write=28.835
              ->  Append  (cost=50588.55..428953.50 rows=1011 width=0) (actual time=313.541..433.629 rows=1001 loops=1)
                    Buffers: shared hit=126380, temp read=10 written=2949
                    I/O Timings: temp read=0.029 write=28.835
                    ->  Subquery Scan on "*SELECT* 1"  (cost=50588.55..51087.20 rows=1001 width=0) (actual time=313.539..433.436 rows=1001 loops=1)
                          Buffers: shared hit=126380, temp read=10 written=2949
                          I/O Timings: temp read=0.029 write=28.835
                          ->  Limit  (cost=50588.55..51077.19 rows=1001 width=4) (actual time=313.538..433.233 rows=1001 loops=1)
                                Buffers: shared hit=126380, temp read=10 written=2949
                                I/O Timings: temp read=0.029 write=28.835
                                ->  Result  (cost=50588.55..59975.23 rows=19229 width=4) (actual time=313.537..433.076 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=126380, temp read=10 written=2949
                                      I/O Timings: temp read=0.029 write=28.835
                                      ->  Hash Semi Join  (cost=50588.55..59975.23 rows=19229 width=0) (actual time=313.531..432.823 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=126380, temp read=10 written=2949
                                            I/O Timings: temp read=0.029 write=28.835
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.028..64.507 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=310.990..310.991 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=17.604
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..43367.12 rows=373429 width=32) (actual time=44.680..231.197 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=50588.55..53196.65 rows=5 width=0) (never executed)
                          ->  Limit  (cost=50588.55..53196.60 rows=5 width=4) (never executed)
                                ->  Result  (cost=50588.55..53196.60 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=50588.55..53196.60 rows=5 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=373429 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=50588.55..53187.61 rows=4 width=0) (never executed)
                          ->  Limit  (cost=50588.55..53187.57 rows=4 width=4) (never executed)
                                ->  Result  (cost=50588.55..53187.57 rows=4 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=50588.55..53187.57 rows=4 width=0) (never executed)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=373429 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=50611.92..271476.99 rows=1 width=0) (never executed)
                          ->  Limit  (cost=50611.92..271476.98 rows=1 width=4) (never executed)
                                ->  Result  (cost=50611.92..271476.98 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=50611.92..271476.98 rows=1 width=0) (never executed)
                                            Hash Cond: (r_3.id = v_3.record_id)
                                            ->  Hash Semi Join  (cost=23.93..218333.98 rows=1 width=48) (never executed)
                                                  Hash Cond: (r_3.id = acl_share.record_id)
                                                  ->  Seq Scan on records r_3  (cost=0.00..217947.19 rows=138232 width=32) (never executed)
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                  ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=373429 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.162 ms
Execution Time: 451.771 ms
```

### Cell spill, unbudgeted count: a50, pw = 1 and pk < 5 (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=630139.04..630139.05 rows=1 width=8) (actual time=1880.740..1880.761 rows=1 loops=1)
  Buffers: shared hit=194696, temp read=8687 written=16288
  I/O Timings: temp read=16.383 write=139.165
  ->  Limit  (cost=100443.45..630126.53 rows=1001 width=4) (actual time=950.320..1880.729 rows=100 loops=1)
        Buffers: shared hit=194696, temp read=8687 written=16288
        I/O Timings: temp read=16.383 write=139.165
        ->  Result  (cost=100443.45..635418.06 rows=1011 width=4) (actual time=950.319..1880.708 rows=100 loops=1)
              Buffers: shared hit=194696, temp read=8687 written=16288
              I/O Timings: temp read=16.383 write=139.165
              ->  Append  (cost=100443.45..635407.95 rows=1011 width=0) (actual time=950.318..1880.682 rows=100 loops=1)
                    Buffers: shared hit=194696, temp read=8687 written=16288
                    I/O Timings: temp read=16.383 write=139.165
                    ->  Subquery Scan on "*SELECT* 1"  (cost=100443.45..101088.84 rows=1001 width=0) (actual time=950.317..971.409 rows=100 loops=1)
                          Buffers: shared hit=143454, temp read=8687 written=10075
                          I/O Timings: temp read=16.383 write=81.909
                          ->  Limit  (cost=100443.45..101078.83 rows=1001 width=4) (actual time=950.315..971.378 rows=100 loops=1)
                                Buffers: shared hit=143454, temp read=8687 written=10075
                                I/O Timings: temp read=16.383 write=81.909
                                ->  Result  (cost=100443.45..112648.83 rows=19229 width=4) (actual time=950.314..971.356 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=143454, temp read=8687 written=10075
                                      I/O Timings: temp read=16.383 write=81.909
                                      ->  Hash Semi Join  (cost=100443.45..112648.83 rows=19229 width=0) (actual time=950.306..971.316 rows=100 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=143454, temp read=8687 written=10075
                                            I/O Timings: temp read=16.383 write=81.909
                                            ->  Hash Semi Join  (cost=49855.46..59007.18 rows=19229 width=48) (actual time=525.756..694.713 rows=100 loops=1)
                                                  Hash Cond: (r.id = v_1.record_id)
                                                  Buffers: shared hit=126380, temp read=7920 written=7920
                                                  I/O Timings: temp read=14.955 write=65.305
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.054..62.736 rows=200379 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109306
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (actual time=394.257..394.258 rows=702464 loops=1)
                                                        Buckets: 2048 (originally 2048)  Batches: 512 (originally 256)  Memory Usage: 113kB
                                                        Buffers: shared hit=17074, temp written=4235
                                                        I/O Timings: temp write=38.202
                                                        ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335503 width=32) (actual time=76.898..219.050 rows=702464 loops=1)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                              Rows Removed by Filter: 800000
                                                              Buffers: shared hit=17074
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=252.686..252.687 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=13.963
                                                  ->  Seq Scan on rfv_num v  (cost=0.00..43367.12 rows=373429 width=32) (actual time=42.521..185.758 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=100443.45..105347.60 rows=5 width=0) (actual time=354.602..354.605 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=2071
                          I/O Timings: temp write=25.317
                          ->  Limit  (cost=100443.45..105347.55 rows=5 width=4) (actual time=354.599..354.602 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=2071
                                I/O Timings: temp write=25.317
                                ->  Result  (cost=100443.45..105347.55 rows=5 width=4) (actual time=354.595..354.598 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=17078, temp written=2071
                                      I/O Timings: temp write=25.317
                                      ->  Hash Semi Join  (cost=100443.45..105347.55 rows=5 width=0) (actual time=354.574..354.576 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=17078, temp written=2071
                                            I/O Timings: temp write=25.317
                                            ->  Hash Semi Join  (cost=49855.46..52204.52 rows=5 width=48) (actual time=0.050..0.052 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_3.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.049..0.049 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=343.891..343.891 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=23.302
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=373429 width=32) (actual time=51.960..245.740 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 3"  (cost=100443.45..105338.56 rows=4 width=0) (actual time=300.379..300.382 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=2071
                          I/O Timings: temp write=17.678
                          ->  Limit  (cost=100443.45..105338.52 rows=4 width=4) (actual time=300.376..300.379 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=2071
                                I/O Timings: temp write=17.678
                                ->  Result  (cost=100443.45..105338.52 rows=4 width=4) (actual time=300.372..300.375 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=17078, temp written=2071
                                      I/O Timings: temp write=17.678
                                      ->  Hash Semi Join  (cost=100443.45..105338.52 rows=4 width=0) (actual time=300.352..300.354 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=17078, temp written=2071
                                            I/O Timings: temp write=17.678
                                            ->  Hash Semi Join  (cost=49855.46..52195.49 rows=4 width=48) (actual time=0.030..0.031 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_5.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.029..0.029 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_5  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=292.493..292.494 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=16.290
                                                  ->  Seq Scan on rfv_num v_4  (cost=0.00..43367.12 rows=373429 width=32) (actual time=53.321..212.188 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 4"  (cost=100466.83..323627.90 rows=1 width=0) (actual time=254.247..254.252 rows=0 loops=1)
                          Buffers: shared hit=17086, temp written=2071
                          I/O Timings: temp write=14.261
                          ->  Limit  (cost=100466.83..323627.89 rows=1 width=4) (actual time=254.245..254.250 rows=0 loops=1)
                                Buffers: shared hit=17086, temp written=2071
                                I/O Timings: temp write=14.261
                                ->  Result  (cost=100466.83..323627.89 rows=1 width=4) (actual time=254.243..254.248 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=17086, temp written=2071
                                      I/O Timings: temp write=14.261
                                      ->  Hash Semi Join  (cost=100466.83..323627.89 rows=1 width=0) (actual time=254.228..254.233 rows=0 loops=1)
                                            Hash Cond: (r_3.id = v_6.record_id)
                                            Buffers: shared hit=17086, temp written=2071
                                            I/O Timings: temp write=14.261
                                            ->  Hash Semi Join  (cost=49878.84..270484.89 rows=1 width=64) (actual time=0.093..0.096 rows=0 loops=1)
                                                  Hash Cond: (r_3.id = v_7.record_id)
                                                  Buffers: shared hit=12
                                                  ->  Hash Semi Join  (cost=23.93..218333.98 rows=1 width=48) (actual time=0.092..0.095 rows=0 loops=1)
                                                        Hash Cond: (r_3.id = acl_share.record_id)
                                                        Buffers: shared hit=12
                                                        ->  Seq Scan on records r_3  (cost=0.00..217947.19 rows=138232 width=32) (actual time=0.058..0.059 rows=1 loops=1)
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                              Rows Removed by Filter: 33
                                                              Buffers: shared hit=6
                                                        ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.025..0.027 rows=0 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                              Buffers: shared hit=6
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.024..0.026 rows=0 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                    Buffers: shared hit=6
                                                                    ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                          Buffers: shared hit=6
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                                Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_7  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=246.918..246.918 rows=311871 loops=1)
                                                  Buckets: 2048  Batches: 256  Memory Usage: 93kB
                                                  Buffers: shared hit=17074, temp written=1816
                                                  I/O Timings: temp write=12.960
                                                  ->  Seq Scan on rfv_num v_6  (cost=0.00..43367.12 rows=373429 width=32) (actual time=43.683..181.487 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 1.845 ms
Execution Time: 1881.153 ms
```

### Cell spill, unbudgeted count: a50, three broad fields (enable_mergejoin = off, enable_nestloop = off, max_parallel_workers_per_gather = 0, work_mem = 64kB)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=478441.79..478441.80 rows=1 width=8) (actual time=654.877..654.895 rows=1 loops=1)
  Buffers: shared hit=124701, temp read=5005 written=6581
  I/O Timings: temp read=13.569 write=55.377
  ->  Limit  (cost=61011.72..478429.28 rows=1001 width=4) (actual time=326.038..654.732 rows=1001 loops=1)
        Buffers: shared hit=124701, temp read=5005 written=6581
        I/O Timings: temp read=13.569 write=55.377
        ->  Result  (cost=61011.72..479680.28 rows=1004 width=4) (actual time=326.036..654.538 rows=1001 loops=1)
              Buffers: shared hit=124701, temp read=5005 written=6581
              I/O Timings: temp read=13.569 write=55.377
              ->  Append  (cost=61011.72..479670.24 rows=1004 width=0) (actual time=326.036..654.374 rows=1001 loops=1)
                    Buffers: shared hit=124701, temp read=5005 written=6581
                    I/O Timings: temp read=13.569 write=55.377
                    ->  Subquery Scan on "*SELECT* 1"  (cost=61011.72..68059.60 rows=1001 width=0) (actual time=326.035..654.235 rows=1001 loops=1)
                          Buffers: shared hit=124701, temp read=5005 written=6581
                          I/O Timings: temp read=13.569 write=55.377
                          ->  Limit  (cost=61011.72..68049.59 rows=1001 width=4) (actual time=326.033..654.081 rows=1001 loops=1)
                                Buffers: shared hit=124701, temp read=5005 written=6581
                                I/O Timings: temp read=13.569 write=55.377
                                ->  Result  (cost=61011.72..71199.40 rows=1449 width=4) (actual time=326.032..653.961 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=124701, temp read=5005 written=6581
                                      I/O Timings: temp read=13.569 write=55.377
                                      ->  Hash Semi Join  (cost=61011.72..71199.40 rows=1449 width=0) (actual time=326.021..653.774 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=124701, temp read=5005 written=6581
                                            I/O Timings: temp read=13.569 write=55.377
                                            ->  Hash Semi Join  (cost=47022.12..55802.94 rows=1454 width=64) (actual time=214.165..522.343 rows=50162 loops=1)
                                                  Hash Cond: (r.id = v_2.record_id)
                                                  Buffers: shared hit=120577, temp read=4934 written=4934
                                                  I/O Timings: temp read=13.423 write=42.386
                                                  ->  Hash Semi Join  (cost=32001.96..39271.48 rows=5759 width=48) (actual time=94.876..306.878 rows=100217 loops=1)
                                                        Hash Cond: (r.id = v_1.record_id)
                                                        Buffers: shared hit=116453, temp read=2728 written=2728
                                                        I/O Timings: temp read=8.533 write=23.174
                                                        ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.043..57.203 rows=200379 loops=1)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=109306
                                                        ->  Hash  (cost=30222.26..30222.26 rows=92012 width=32) (actual time=92.927..92.928 rows=155850 loops=1)
                                                              Buckets: 2048 (originally 2048)  Batches: 128 (originally 64)  Memory Usage: 113kB
                                                              Buffers: shared hit=7147, temp written=899
                                                              I/O Timings: temp write=7.555
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..30222.26 rows=92012 width=32) (actual time=0.019..54.656 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                                    Heap Fetches: 155850
                                                                    Buffers: shared hit=7147
                                                  ->  Hash  (cost=11124.00..11124.00 rows=201453 width=32) (actual time=119.270..119.271 rows=199730 loops=1)
                                                        Buckets: 2048  Batches: 128  Memory Usage: 108kB
                                                        Buffers: shared hit=4124, temp written=1190
                                                        I/O Timings: temp write=9.981
                                                        ->  Seq Scan on rfv_text v_2  (cost=0.00..11124.00 rows=201453 width=32) (actual time=0.014..71.171 rows=199730 loops=1)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid) AND (value = 'x'::text COLLATE "C"))
                                                              Rows Removed by Filter: 200270
                                                              Buffers: shared hit=4124
                                            ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (actual time=111.831..111.831 rows=199887 loops=1)
                                                  Buckets: 2048  Batches: 128  Memory Usage: 115kB
                                                  Buffers: shared hit=4124, temp written=1189
                                                  I/O Timings: temp write=10.283
                                                  ->  Seq Scan on rfv_bool v  (cost=0.00..10124.00 rows=199887 width=32) (actual time=0.014..61.316 rows=199887 loops=1)
                                                        Filter: (value AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                                                        Rows Removed by Filter: 200113
                                                        Buffers: shared hit=4124
                    ->  Subquery Scan on "*SELECT* 2"  (cost=61011.72..64444.73 rows=1 width=0) (never executed)
                          ->  Limit  (cost=61011.72..64444.72 rows=1 width=4) (never executed)
                                ->  Result  (cost=61011.72..64444.72 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=61011.72..64444.72 rows=1 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_3.record_id)
                                            ->  Hash Semi Join  (cost=47022.12..49086.12 rows=1 width=64) (never executed)
                                                  Hash Cond: (r_1.id = v_5.record_id)
                                                  ->  Hash Semi Join  (cost=32001.96..32685.95 rows=1 width=48) (never executed)
                                                        Hash Cond: (r_1.id = v_4.record_id)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        ->  Hash  (cost=30222.26..30222.26 rows=92012 width=32) (never executed)
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..30222.26 rows=92012 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                                    Heap Fetches: 0
                                                  ->  Hash  (cost=11124.00..11124.00 rows=201453 width=32) (never executed)
                                                        ->  Seq Scan on rfv_text v_5  (cost=0.00..11124.00 rows=201453 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid) AND (value = 'x'::text COLLATE "C"))
                                            ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (never executed)
                                                  ->  Seq Scan on rfv_bool v_3  (cost=0.00..10124.00 rows=199887 width=32) (never executed)
                                                        Filter: (value AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=61011.72..64435.72 rows=1 width=0) (never executed)
                          ->  Limit  (cost=61011.72..64435.71 rows=1 width=4) (never executed)
                                ->  Result  (cost=61011.72..64435.71 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=61011.72..64435.71 rows=1 width=0) (never executed)
                                            Hash Cond: (r_2.id = v_6.record_id)
                                            ->  Hash Semi Join  (cost=47022.12..49077.11 rows=1 width=64) (never executed)
                                                  Hash Cond: (r_2.id = v_8.record_id)
                                                  ->  Hash Semi Join  (cost=32001.96..32676.94 rows=1 width=48) (never executed)
                                                        Hash Cond: (r_2.id = v_7.record_id)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                        ->  Hash  (cost=30222.26..30222.26 rows=92012 width=32) (never executed)
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..30222.26 rows=92012 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                                    Heap Fetches: 0
                                                  ->  Hash  (cost=11124.00..11124.00 rows=201453 width=32) (never executed)
                                                        ->  Seq Scan on rfv_text v_8  (cost=0.00..11124.00 rows=201453 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid) AND (value = 'x'::text COLLATE "C"))
                                            ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (never executed)
                                                  ->  Seq Scan on rfv_bool v_6  (cost=0.00..10124.00 rows=199887 width=32) (never executed)
                                                        Filter: (value AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=61035.10..282725.17 rows=1 width=0) (never executed)
                          ->  Limit  (cost=61035.10..282725.16 rows=1 width=4) (never executed)
                                ->  Result  (cost=61035.10..282725.16 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=61035.10..282725.16 rows=1 width=0) (never executed)
                                            Hash Cond: (v_9.record_id = v_10.record_id)
                                            ->  Hash Semi Join  (cost=29033.68..250092.74 rows=1 width=80) (never executed)
                                                  Hash Cond: (r_3.id = v_9.record_id)
                                                  ->  Hash Semi Join  (cost=15044.09..234734.15 rows=1 width=64) (never executed)
                                                        Hash Cond: (r_3.id = v_11.record_id)
                                                        ->  Hash Semi Join  (cost=23.93..218333.98 rows=1 width=48) (never executed)
                                                              Hash Cond: (r_3.id = acl_share.record_id)
                                                              ->  Seq Scan on records r_3  (cost=0.00..217947.19 rows=138232 width=32) (never executed)
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                              ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        ->  Hash  (cost=11124.00..11124.00 rows=201453 width=32) (never executed)
                                                              ->  Seq Scan on rfv_text v_11  (cost=0.00..11124.00 rows=201453 width=32) (never executed)
                                                                    Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid) AND (value = 'x'::text COLLATE "C"))
                                                  ->  Hash  (cost=10124.00..10124.00 rows=199887 width=32) (never executed)
                                                        ->  Seq Scan on rfv_bool v_9  (cost=0.00..10124.00 rows=199887 width=32) (never executed)
                                                              Filter: (value AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                                            ->  Hash  (cost=30222.26..30222.26 rows=92012 width=32) (never executed)
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_10  (cost=0.55..30222.26 rows=92012 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                        Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_mergejoin = 'off', enable_nestloop = 'off', max_parallel_workers_per_gather = '0', work_mem = '64kB'
Planning Time: 3.459 ms
Execution Time: 663.655 ms
```

### Cell parallel, unbudgeted count: a50, pk = 3 (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=27585.25..27585.26 rows=1 width=8) (actual time=105.560..114.908 rows=1 loops=1)
  Buffers: shared hit=9514
  ->  Limit  (cost=26190.41..27572.74 rows=1001 width=4) (actual time=100.624..114.696 rows=1001 loops=1)
        Buffers: shared hit=9514
        ->  Result  (cost=26190.41..27576.88 rows=1004 width=4) (actual time=100.621..114.419 rows=1001 loops=1)
              Buffers: shared hit=9514
              ->  Append  (cost=26190.41..27566.84 rows=1004 width=0) (actual time=100.620..114.087 rows=1001 loops=1)
                    Buffers: shared hit=9514
                    ->  Subquery Scan on "*SELECT* 1"  (cost=26190.41..27260.97 rows=1001 width=0) (actual time=100.618..113.823 rows=1001 loops=1)
                          Buffers: shared hit=9514
                          ->  Limit  (cost=26190.41..27250.96 rows=1001 width=4) (actual time=100.616..113.512 rows=1001 loops=1)
                                Buffers: shared hit=9514
                                ->  Gather  (cost=26190.41..32292.00 rows=5759 width=4) (actual time=100.614..113.317 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=9514
                                      ->  Result  (cost=25190.41..30716.10 rows=2400 width=4) (actual time=91.521..95.609 rows=739 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            Buffers: shared hit=9514
                                            ->  Parallel Hash Semi Join  (cost=25190.41..30716.10 rows=2400 width=0) (actual time=91.505..95.361 rows=739 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=9514
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5441.94 rows=20801 width=32) (actual time=0.245..1.920 rows=1503 loops=3)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=2310
                                                  ->  Parallel Hash  (cost=24710.63..24710.63 rows=38338 width=32) (actual time=90.348..90.350 rows=51950 loops=3)
                                                        Buckets: 262144 (originally 131072)  Batches: 1 (originally 1)  Memory Usage: 0kB
                                                        Buffers: shared hit=7146
                                                        ->  Parallel Bitmap Heap Scan on rfv_num v  (cost=6965.71..24710.63 rows=38338 width=32) (actual time=19.661..50.725 rows=51950 loops=3)
                                                              Recheck Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                              Heap Blocks: exact=1779
                                                              Buffers: shared hit=7146
                                                              ->  Bitmap Index Scan on rfv_num_value  (cost=0.00..6942.70 rows=92012 width=0) (actual time=25.479..25.480 rows=155850 loops=1)
                                                                    Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision))
                                                                    Buffers: shared hit=2600
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..144.57 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..144.56 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..144.56 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..144.56 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..122.39 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.11..122.38 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.11..122.38 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..122.38 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.79..33.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.79..33.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r_3.id) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.026 ms
Execution Time: 115.170 ms
```

### Cell parallel, unbudgeted count: a6, three broad fields (server settings)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10776.16..10776.17 rows=1 width=8) (actual time=264.316..291.247 rows=1 loops=1)
  Buffers: shared hit=58454
  ->  Limit  (cost=8666.49..10773.98 rows=174 width=4) (actual time=157.842..290.825 rows=1001 loops=1)
        Buffers: shared hit=58454
        ->  Result  (cost=8666.49..10773.98 rows=174 width=4) (actual time=157.840..290.550 rows=1001 loops=1)
              Buffers: shared hit=58454
              ->  Append  (cost=8666.49..10772.24 rows=174 width=0) (actual time=157.838..290.124 rows=1001 loops=1)
                    Buffers: shared hit=58454
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10422.49 rows=171 width=0) (actual time=157.836..289.793 rows=1001 loops=1)
                          Buffers: shared hit=58454
                          ->  Limit  (cost=8666.49..10420.78 rows=171 width=4) (actual time=157.833..289.332 rows=1001 loops=1)
                                Buffers: shared hit=58454
                                ->  Gather  (cost=8666.49..10420.78 rows=171 width=4) (actual time=157.831..289.051 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=58454
                                      ->  Result  (cost=7666.49..9403.68 rows=101 width=4) (actual time=151.937..256.874 rows=780 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            Buffers: shared hit=58454
                                            ->  Nested Loop Semi Join  (cost=7666.49..9403.68 rows=101 width=0) (actual time=151.914..256.464 rows=780 loops=2)
                                                  Join Filter: (v.record_id = v_1.record_id)
                                                  Buffers: shared hit=58454
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8892.07 rows=438 width=64) (actual time=151.807..220.325 rows=1931 loops=2)
                                                        Buffers: shared hit=43005
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8360.32 rows=871 width=48) (actual time=151.730..172.466 rows=3866 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=12076
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..676.43 rows=3474 width=32) (actual time=0.118..6.501 rows=7658 loops=2)
                                                                    Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=7923
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=150.463..150.465 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=0.025..68.893 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=7732)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v.record_id) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=30929
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.43..1.16 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=3862)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_2.record_id) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=15449
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..177.00 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..176.99 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..176.99 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..176.99 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..176.36 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..174.76 rows=2 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=16 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '49d2eb81-95cd-4e20-b268-84309b755544'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_4.record_id) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_5.record_id) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..136.87 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..136.86 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..136.86 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..136.86 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..136.23 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..135.43 rows=1 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '49d2eb81-95cd-4e20-b268-84309b755544'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '49d2eb81-95cd-4e20-b268-84309b755544'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.79 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_7.record_id) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_8.record_id) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..35.01 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..35.00 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..35.00 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..35.00 rows=1 width=0) (never executed)
                                            Join Filter: (v_9.record_id = v_10.record_id)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.82 rows=1 width=80) (never executed)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.21 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '49d2eb81-95cd-4e20-b268-84309b755544'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '49d2eb81-95cd-4e20-b268-84309b755544'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '49d2eb81-95cd-4e20-b268-84309b755544'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,5ddd0712-afc8-4d67-bad0-65c7f4d4e142}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '49d2eb81-95cd-4e20-b268-84309b755544'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '49d2eb81-95cd-4e20-b268-84309b755544'::uuid)) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                        ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.62 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r_3.id) AND (field_id = '289990c1-ca8d-4f76-b958-68e809670b03'::uuid))
                                                              Filter: value
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.61 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_9.record_id) AND (field_id = '99fb238d-51e6-4914-93b0-25ce73f73f2a'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.43..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = v_11.record_id) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.908 ms
Execution Time: 291.681 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 (parallel_setup_cost = 0, parallel_tuple_cost = 0)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9056.97..9056.98 rows=1 width=8) (actual time=932.579..932.590 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9044.46 rows=1001 width=4) (actual time=4.943..932.367 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9134.80 rows=1011 width=4) (actual time=4.941..932.258 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9124.69 rows=1011 width=0) (actual time=4.940..932.105 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8821.51 rows=1001 width=0) (actual time=4.938..931.885 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8811.50 rows=1001 width=4) (actual time=4.937..931.760 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169249.06 rows=19229 width=4) (actual time=4.936..931.670 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169249.06 rows=19229 width=0) (actual time=4.927..931.512 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.035..62.898 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r.id) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..143.06 rows=5 width=0) (actual time=0.041..0.043 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..143.01 rows=5 width=4) (actual time=0.040..0.041 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..143.01 rows=5 width=4) (actual time=0.037..0.038 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..143.01 rows=5 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.022..0.022 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..121.14 rows=4 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..121.10 rows=4 width=4) (actual time=0.019..0.020 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..121.10 rows=4 width=4) (actual time=0.018..0.019 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..121.10 rows=4 width=0) (actual time=0.014..0.015 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.014..0.014 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.92 rows=1 width=0) (actual time=0.032..0.035 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.91 rows=1 width=4) (actual time=0.031..0.034 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.91 rows=1 width=4) (actual time=0.030..0.033 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.91 rows=1 width=0) (actual time=0.026..0.029 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.026..0.028 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.025..0.027 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.021..0.023 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.008..0.009 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.50 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r_3.id) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', parallel_setup_cost = '0', parallel_tuple_cost = '0'
Planning Time: 1.708 ms
Execution Time: 932.721 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 (max_parallel_workers_per_gather = 1, parallel_setup_cost = 0, parallel_tuple_cost = 0)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9056.97..9056.98 rows=1 width=8) (actual time=708.967..708.974 rows=1 loops=1)
  Buffers: shared hit=710557
  ->  Limit  (cost=0.99..9044.46 rows=1001 width=4) (actual time=4.908..708.766 rows=100 loops=1)
        Buffers: shared hit=710557
        ->  Result  (cost=0.99..9134.80 rows=1011 width=4) (actual time=4.907..708.662 rows=100 loops=1)
              Buffers: shared hit=710557
              ->  Append  (cost=0.99..9124.69 rows=1011 width=0) (actual time=4.905..708.558 rows=100 loops=1)
                    Buffers: shared hit=710557
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..8821.51 rows=1001 width=0) (actual time=4.904..708.362 rows=100 loops=1)
                          Buffers: shared hit=710543
                          ->  Limit  (cost=0.99..8811.50 rows=1001 width=4) (actual time=4.903..708.251 rows=100 loops=1)
                                Buffers: shared hit=710543
                                ->  Result  (cost=0.99..169249.06 rows=19229 width=4) (actual time=4.902..708.161 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=710543
                                      ->  Nested Loop Semi Join  (cost=0.99..169249.06 rows=19229 width=0) (actual time=4.895..708.000 rows=100 loops=1)
                                            Buffers: shared hit=710543
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5733.16 rows=49923 width=32) (actual time=0.033..48.028 rows=200379 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=109306
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..3.31 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=200379)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r.id) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid))
                                                  Filter: (value = '1'::double precision)
                                                  Buffers: shared hit=601237
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..143.06 rows=5 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..143.01 rows=5 width=4) (actual time=0.029..0.030 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..143.01 rows=5 width=4) (actual time=0.027..0.028 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..143.01 rows=5 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.015..0.015 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..7.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..121.14 rows=4 width=0) (actual time=0.043..0.044 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..121.10 rows=4 width=4) (actual time=0.042..0.043 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..121.10 rows=4 width=4) (actual time=0.042..0.042 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..121.10 rows=4 width=0) (actual time=0.039..0.039 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.039..0.039 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..7.78 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.79..33.92 rows=1 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.79..33.91 rows=1 width=4) (actual time=0.024..0.025 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.79..33.91 rows=1 width=4) (actual time=0.023..0.025 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.79..33.91 rows=1 width=0) (actual time=0.021..0.023 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.021..0.022 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.020..0.021 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.43..1.50 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (record_id = r_3.id) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid))
                                                  Filter: (value = '1'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', max_parallel_workers_per_gather = '1', parallel_setup_cost = '0', parallel_tuple_cost = '0'
Planning Time: 1.649 ms
Execution Time: 709.091 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 (enable_nestloop = off)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=373556.70..373556.71 rows=1 width=8) (actual time=698.995..735.509 rows=1 loops=1)
  Buffers: shared hit=143615, temp read=14078 written=14188
  I/O Timings: temp read=37.010 write=208.287
  ->  Limit  (cost=31733.44..373544.18 rows=1001 width=4) (actual time=325.473..735.413 rows=100 loops=1)
        Buffers: shared hit=143615, temp read=14078 written=14188
        I/O Timings: temp read=37.010 write=208.287
        ->  Result  (cost=31733.44..376958.88 rows=1011 width=4) (actual time=325.471..735.381 rows=100 loops=1)
              Buffers: shared hit=143615, temp read=14078 written=14188
              I/O Timings: temp read=37.010 write=208.287
              ->  Append  (cost=31733.44..376948.77 rows=1011 width=0) (actual time=325.470..735.335 rows=100 loops=1)
                    Buffers: shared hit=143615, temp read=14078 written=14188
                    I/O Timings: temp read=37.010 write=208.287
                    ->  Subquery Scan on "*SELECT* 1"  (cost=31733.44..32199.47 rows=1001 width=0) (actual time=325.469..406.962 rows=100 loops=1)
                          Buffers: shared hit=126452, temp read=7683 written=7764
                          I/O Timings: temp read=20.879 write=99.698
                          ->  Limit  (cost=31733.44..32189.46 rows=1001 width=4) (actual time=325.467..406.916 rows=100 loops=1)
                                Buffers: shared hit=126452, temp read=7683 written=7764
                                I/O Timings: temp read=20.879 write=99.698
                                ->  Gather  (cost=31733.44..40493.49 rows=19229 width=4) (actual time=325.465..406.878 rows=100 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=126452, temp read=7683 written=7764
                                      I/O Timings: temp read=20.879 write=99.698
                                      ->  Result  (cost=30733.44..37570.59 rows=8012 width=4) (actual time=334.317..377.862 rows=33 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            Buffers: shared hit=126452, temp read=7683 written=7764
                                            I/O Timings: temp read=20.879 write=99.698
                                            ->  Parallel Hash Semi Join  (cost=30733.44..37570.59 rows=8012 width=0) (actual time=334.306..377.821 rows=33 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=126452, temp read=7683 written=7764
                                                  I/O Timings: temp read=20.879 write=99.698
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5441.94 rows=20801 width=32) (actual time=0.171..56.706 rows=66793 loops=3)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109320
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=139793 width=32) (actual time=237.039..237.041 rows=234155 loops=3)
                                                        Buckets: 131072 (originally 131072)  Batches: 8 (originally 4)  Memory Usage: 6592kB
                                                        Buffers: shared hit=17074, temp read=2484 written=6444
                                                        I/O Timings: temp read=7.105 write=78.806
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..28029.47 rows=139793 width=32) (actual time=44.924..115.819 rows=234155 loops=3)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                              Rows Removed by Filter: 266667
                                                              Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=49855.47..52204.58 rows=5 width=0) (actual time=0.057..0.063 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49855.47..52204.53 rows=5 width=4) (actual time=0.056..0.060 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49855.47..52204.53 rows=5 width=4) (actual time=0.054..0.058 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49855.47..52204.53 rows=5 width=0) (actual time=0.038..0.042 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.037..0.038 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=49855.47..52195.54 rows=4 width=0) (actual time=0.021..0.024 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=49855.47..52195.50 rows=4 width=4) (actual time=0.020..0.023 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=49855.47..52195.50 rows=4 width=4) (actual time=0.019..0.021 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=4
                                      ->  Hash Semi Join  (cost=49855.47..52195.50 rows=4 width=0) (actual time=0.016..0.018 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.016..0.016 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                        Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=31756.82..240344.12 rows=1 width=0) (actual time=314.034..328.240 rows=0 loops=1)
                          Buffers: shared hit=17155, temp read=6395 written=6424
                          I/O Timings: temp read=16.131 write=108.589
                          ->  Limit  (cost=31756.82..240344.11 rows=1 width=4) (actual time=314.033..328.237 rows=0 loops=1)
                                Buffers: shared hit=17155, temp read=6395 written=6424
                                I/O Timings: temp read=16.131 write=108.589
                                ->  Gather  (cost=31756.82..240344.11 rows=1 width=4) (actual time=314.030..328.232 rows=0 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=17155, temp read=6395 written=6424
                                      I/O Timings: temp read=16.131 write=108.589
                                      ->  Result  (cost=30756.82..239344.01 rows=1 width=4) (actual time=305.294..305.303 rows=0 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            Buffers: shared hit=17155, temp read=6395 written=6424
                                            I/O Timings: temp read=16.131 write=108.589
                                            ->  Parallel Hash Semi Join  (cost=30756.82..239344.01 rows=1 width=0) (actual time=305.283..305.291 rows=0 loops=3)
                                                  Hash Cond: (r_3.id = v_3.record_id)
                                                  Buffers: shared hit=17155, temp read=6395 written=6424
                                                  I/O Timings: temp read=16.131 write=108.589
                                                  ->  Hash Semi Join  (cost=23.93..207653.12 rows=1 width=48) (actual time=0.104..0.108 rows=0 loops=3)
                                                        Hash Cond: (r_3.id = acl_share.record_id)
                                                        Buffers: shared hit=23
                                                        ->  Parallel Seq Scan on records r_3  (cost=0.00..207478.00 rows=57597 width=32) (actual time=0.036..0.037 rows=1 loops=3)
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=3
                                                        ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.055..0.058 rows=0 loops=3)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                              Buffers: shared hit=20
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.054..0.057 rows=0 loops=3)
                                                                    Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                    Buffers: shared hit=20
                                                                    ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.053..0.054 rows=0 loops=3)
                                                                          Buffers: shared hit=20
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.035..0.035 rows=0 loops=3)
                                                                                Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                                Buffers: shared hit=8
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.016..0.017 rows=0 loops=3)
                                                                                Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                                Buffers: shared hit=12
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=139793 width=32) (actual time=264.322..264.323 rows=234155 loops=3)
                                                        Buckets: 131072 (originally 131072)  Batches: 8 (originally 4)  Memory Usage: 6592kB
                                                        Buffers: shared hit=17074, temp read=2481 written=6424
                                                        I/O Timings: temp read=7.633 write=108.589
                                                        ->  Parallel Seq Scan on rfv_num v_3  (cost=0.00..28029.47 rows=139793 width=32) (actual time=41.811..120.491 rows=234155 loops=3)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                              Rows Removed by Filter: 266667
                                                              Buffers: shared hit=17074
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off'
Planning Time: 2.335 ms
Execution Time: 735.730 ms
```

### Cell parallel, unbudgeted count: a50, pk < 5 (enable_nestloop = off)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=376241.74..376241.75 rows=1 width=8) (actual time=267.043..281.913 rows=1 loops=1)
  Buffers: shared hit=126510, temp read=1007 written=2800
  I/O Timings: temp read=4.865 write=29.319
  ->  Limit  (cost=32038.97..376229.22 rows=1001 width=4) (actual time=265.856..281.680 rows=1001 loops=1)
        Buffers: shared hit=126510, temp read=1007 written=2800
        I/O Timings: temp read=4.865 write=29.319
        ->  Result  (cost=32038.97..379667.69 rows=1011 width=4) (actual time=265.854..281.540 rows=1001 loops=1)
              Buffers: shared hit=126510, temp read=1007 written=2800
              I/O Timings: temp read=4.865 write=29.319
              ->  Append  (cost=32038.97..379657.58 rows=1011 width=0) (actual time=265.853..281.331 rows=1001 loops=1)
                    Buffers: shared hit=126510, temp read=1007 written=2800
                    I/O Timings: temp read=4.865 write=29.319
                    ->  Subquery Scan on "*SELECT* 1"  (cost=32038.97..32510.62 rows=1001 width=0) (actual time=265.851..281.183 rows=1001 loops=1)
                          Buffers: shared hit=126510, temp read=1007 written=2800
                          I/O Timings: temp read=4.865 write=29.319
                          ->  Limit  (cost=32038.97..32500.61 rows=1001 width=4) (actual time=265.849..281.023 rows=1001 loops=1)
                                Buffers: shared hit=126510, temp read=1007 written=2800
                                I/O Timings: temp read=4.865 write=29.319
                                ->  Gather  (cost=32038.97..40907.01 rows=19229 width=4) (actual time=265.848..280.892 rows=1001 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=126510, temp read=1007 written=2800
                                      I/O Timings: temp read=4.865 write=29.319
                                      ->  Result  (cost=31038.97..37984.11 rows=8012 width=4) (actual time=252.949..254.143 rows=553 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            Buffers: shared hit=126510, temp read=1007 written=2800
                                            I/O Timings: temp read=4.865 write=29.319
                                            ->  Parallel Hash Semi Join  (cost=31038.97..37984.11 rows=8012 width=0) (actual time=252.939..254.018 rows=553 loops=3)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=126510, temp read=1007 written=2800
                                                  I/O Timings: temp read=4.865 write=29.319
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..5441.94 rows=20801 width=32) (actual time=0.174..67.405 rows=66793 loops=3)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=109378
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=155595 width=32) (actual time=144.165..144.166 rows=103957 loops=3)
                                                        Buckets: 131072  Batches: 4  Memory Usage: 0kB
                                                        Buffers: shared hit=17074, temp written=1504
                                                        I/O Timings: temp write=14.355
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..28029.47 rows=155595 width=32) (actual time=21.780..106.189 rows=103957 loops=3)
                                                              Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                              Rows Removed by Filter: 396864
                                                              Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 2"  (cost=50588.55..53196.65 rows=5 width=0) (never executed)
                          ->  Limit  (cost=50588.55..53196.60 rows=5 width=4) (never executed)
                                ->  Result  (cost=50588.55..53196.60 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=50588.55..53196.60 rows=5 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_1  (cost=0.00..43367.12 rows=373429 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=50588.55..53187.61 rows=4 width=0) (never executed)
                          ->  Limit  (cost=50588.55..53187.57 rows=4 width=4) (never executed)
                                ->  Result  (cost=50588.55..53187.57 rows=4 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      ->  Hash Semi Join  (cost=50588.55..53187.57 rows=4 width=0) (never executed)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (never executed)
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=373429 width=32) (never executed)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=32062.35..240757.65 rows=1 width=0) (never executed)
                          ->  Limit  (cost=32062.35..240757.64 rows=1 width=4) (never executed)
                                ->  Gather  (cost=32062.35..240757.64 rows=1 width=4) (never executed)
                                      Workers Planned: 2
                                      Workers Launched: 0
                                      ->  Result  (cost=31062.35..239757.54 rows=1 width=4) (never executed)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            ->  Parallel Hash Semi Join  (cost=31062.35..239757.54 rows=1 width=0) (never executed)
                                                  Hash Cond: (r_3.id = v_3.record_id)
                                                  ->  Hash Semi Join  (cost=23.93..207653.12 rows=1 width=48) (never executed)
                                                        Hash Cond: (r_3.id = acl_share.record_id)
                                                        ->  Parallel Seq Scan on records r_3  (cost=0.00..207478.00 rows=57597 width=32) (never executed)
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                        ->  Hash  (cost=23.92..23.92 rows=1 width=32) (never executed)
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                    Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                    ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=155595 width=32) (never executed)
                                                        ->  Parallel Seq Scan on rfv_num v_3  (cost=0.00..28029.47 rows=155595 width=32) (never executed)
                                                              Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off'
Planning Time: 1.798 ms
Execution Time: 282.064 ms
```

### Cell parallel, unbudgeted count: a50, pw = 1 and pk < 5 (enable_nestloop = off)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=523041.09..523041.10 rows=1 width=8) (actual time=2183.385..2205.601 rows=1 loops=1)
  Buffers: shared hit=515198, temp read=7880 written=10916
  I/O Timings: temp read=21.292 write=139.905
  ->  Limit  (cost=3161.70..523028.58 rows=1001 width=4) (actual time=0.104..2205.576 rows=100 loops=1)
        Buffers: shared hit=515198, temp read=7880 written=10916
        I/O Timings: temp read=21.292 write=139.905
        ->  Result  (cost=3161.70..528222.06 rows=1011 width=4) (actual time=0.103..2205.559 rows=100 loops=1)
              Buffers: shared hit=515198, temp read=7880 written=10916
              I/O Timings: temp read=21.292 write=139.905
              ->  Append  (cost=3161.70..528211.95 rows=1011 width=0) (actual time=0.102..2205.538 rows=100 loops=1)
                    Buffers: shared hit=515198, temp read=7880 written=10916
                    I/O Timings: temp read=21.292 write=139.905
                    ->  Subquery Scan on "*SELECT* 1"  (cost=3161.70..45072.20 rows=1001 width=0) (actual time=0.102..1155.011 rows=100 loops=1)
                          Buffers: shared hit=446812
                          ->  Limit  (cost=3161.70..45062.19 rows=1001 width=4) (actual time=0.101..1154.992 rows=100 loops=1)
                                Buffers: shared hit=446812
                                ->  Result  (cost=3161.70..808061.28 rows=19229 width=4) (actual time=0.100..1154.975 rows=100 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=446812
                                      ->  Merge Semi Join  (cost=3161.70..808061.28 rows=19229 width=0) (actual time=0.090..1154.943 rows=100 loops=1)
                                            Merge Cond: (v.record_id = v_1.record_id)
                                            Buffers: shared hit=446812
                                            ->  Merge Semi Join  (cost=3161.13..754805.21 rows=19229 width=48) (actual time=0.059..893.853 rows=200376 loops=1)
                                                  Merge Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=427116
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..630958.15 rows=49923 width=32) (actual time=0.039..683.274 rows=200376 loops=1)
                                                        Index Cond: (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                        Rows Removed by Filter: 199621
                                                        Buffers: shared hit=404376
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.43..125715.24 rows=373429 width=32) (actual time=0.018..135.618 rows=311868 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Filter: (value < '5'::double precision)
                                                        Rows Removed by Filter: 88129
                                                        Buffers: shared hit=22740
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..52159.24 rows=335503 width=32) (actual time=0.023..181.070 rows=702464 loops=1)
                                                  Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                  Heap Fetches: 702464
                                                  Buffers: shared hit=19696
                    ->  Subquery Scan on "*SELECT* 2"  (cost=100443.45..105347.60 rows=5 width=0) (actual time=267.593..267.601 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=1486
                          I/O Timings: temp write=11.736
                          ->  Limit  (cost=100443.45..105347.55 rows=5 width=4) (actual time=267.591..267.598 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=1486
                                I/O Timings: temp write=11.736
                                ->  Result  (cost=100443.45..105347.55 rows=5 width=4) (actual time=267.588..267.595 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=17078, temp written=1486
                                      I/O Timings: temp write=11.736
                                      ->  Hash Semi Join  (cost=100443.45..105347.55 rows=5 width=0) (actual time=267.566..267.572 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=17078, temp written=1486
                                            I/O Timings: temp write=11.736
                                            ->  Hash Semi Join  (cost=49855.46..52204.52 rows=5 width=48) (actual time=0.031..0.034 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_3.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.55..53.50 rows=12 width=32) (actual time=0.030..0.030 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_3  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=264.956..264.958 rows=311871 loops=1)
                                                  Buckets: 131072  Batches: 4  Memory Usage: 5898kB
                                                  Buffers: shared hit=17074, temp written=1483
                                                  I/O Timings: temp write=11.657
                                                  ->  Seq Scan on rfv_num v_2  (cost=0.00..43367.12 rows=373429 width=32) (actual time=44.610..195.163 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 3"  (cost=100443.45..105338.56 rows=4 width=0) (actual time=304.906..304.917 rows=0 loops=1)
                          Buffers: shared hit=17078, temp written=1486
                          I/O Timings: temp write=15.741
                          ->  Limit  (cost=100443.45..105338.52 rows=4 width=4) (actual time=304.904..304.915 rows=0 loops=1)
                                Buffers: shared hit=17078, temp written=1486
                                I/O Timings: temp write=15.741
                                ->  Result  (cost=100443.45..105338.52 rows=4 width=4) (actual time=304.901..304.911 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                      Buffers: shared hit=17078, temp written=1486
                                      I/O Timings: temp write=15.741
                                      ->  Hash Semi Join  (cost=100443.45..105338.52 rows=4 width=0) (actual time=304.886..304.895 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=17078, temp written=1486
                                            I/O Timings: temp write=15.741
                                            ->  Hash Semi Join  (cost=49855.46..52195.49 rows=4 width=48) (actual time=0.053..0.060 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_5.record_id)
                                                  Buffers: shared hit=4
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..44.49 rows=10 width=32) (actual time=0.051..0.052 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)))
                                                        Buffers: shared hit=4
                                                  ->  Hash  (cost=43367.12..43367.12 rows=335503 width=32) (never executed)
                                                        ->  Seq Scan on rfv_num v_5  (cost=0.00..43367.12 rows=335503 width=32) (never executed)
                                                              Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                            ->  Hash  (cost=43367.12..43367.12 rows=373429 width=32) (actual time=302.066..302.067 rows=311871 loops=1)
                                                  Buckets: 131072  Batches: 4  Memory Usage: 5898kB
                                                  Buffers: shared hit=17074, temp written=1483
                                                  I/O Timings: temp write=15.672
                                                  ->  Seq Scan on rfv_num v_4  (cost=0.00..43367.12 rows=373429 width=32) (actual time=43.915..214.466 rows=311871 loops=1)
                                                        Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                        Rows Removed by Filter: 1190593
                                                        Buffers: shared hit=17074
                    ->  Subquery Scan on "*SELECT* 4"  (cost=62795.22..272448.53 rows=1 width=0) (actual time=455.794..477.977 rows=0 loops=1)
                          Buffers: shared hit=34230, temp read=7880 written=7944
                          I/O Timings: temp read=21.292 write=112.428
                          ->  Limit  (cost=62795.22..272448.52 rows=1 width=4) (actual time=455.791..477.973 rows=0 loops=1)
                                Buffers: shared hit=34230, temp read=7880 written=7944
                                I/O Timings: temp read=21.292 write=112.428
                                ->  Gather  (cost=62795.22..272448.52 rows=1 width=4) (actual time=455.787..477.967 rows=0 loops=1)
                                      Workers Planned: 2
                                      Workers Launched: 2
                                      Buffers: shared hit=34230, temp read=7880 written=7944
                                      I/O Timings: temp read=21.292 write=112.428
                                      ->  Result  (cost=61795.22..271448.42 rows=1 width=4) (actual time=448.537..448.551 rows=0 loops=3)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid)
                                            Buffers: shared hit=34230, temp read=7880 written=7944
                                            I/O Timings: temp read=21.292 write=112.428
                                            ->  Parallel Hash Semi Join  (cost=61795.22..271448.42 rows=1 width=0) (actual time=448.527..448.540 rows=0 loops=3)
                                                  Hash Cond: (r_3.id = v_6.record_id)
                                                  Buffers: shared hit=34230, temp read=7880 written=7944
                                                  I/O Timings: temp read=21.292 write=112.428
                                                  ->  Parallel Hash Semi Join  (cost=30756.81..239344.00 rows=1 width=64) (actual time=299.836..299.846 rows=0 loops=3)
                                                        Hash Cond: (r_3.id = v_7.record_id)
                                                        Buffers: shared hit=17098, temp read=6391 written=6440
                                                        I/O Timings: temp read=16.782 write=97.179
                                                        ->  Hash Semi Join  (cost=23.93..207653.12 rows=1 width=48) (actual time=0.158..0.164 rows=0 loops=3)
                                                              Hash Cond: (r_3.id = acl_share.record_id)
                                                              Buffers: shared hit=24
                                                              ->  Parallel Seq Scan on records r_3  (cost=0.00..207478.00 rows=57597 width=32) (actual time=0.039..0.039 rows=1 loops=3)
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (entity_definition_id = '01a202bc-3217-76fb-a78e-8bfc112cb475'::uuid))
                                                                    Rows Removed by Filter: 3
                                                                    Buffers: shared hit=4
                                                              ->  Hash  (cost=23.92..23.92 rows=1 width=32) (actual time=0.106..0.109 rows=0 loops=3)
                                                                    Buckets: 1024  Batches: 1  Memory Usage: 8kB
                                                                    Buffers: shared hit=20
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.105..0.108 rows=0 loops=3)
                                                                          Recheck Cond: (((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[]))))
                                                                          Buffers: shared hit=20
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.103..0.105 rows=0 loops=3)
                                                                                Buffers: shared hit=20
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.081..0.081 rows=0 loops=3)
                                                                                      Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = '10845afb-02f8-4fe8-baf3-5fa552103ac5'::uuid))
                                                                                      Buffers: shared hit=8
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.021..0.021 rows=0 loops=3)
                                                                                      Index Cond: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,eb6f8503-0a26-44dd-b0dd-4b4e1809b71f}'::uuid[])))
                                                                                      Buffers: shared hit=12
                                                        ->  Parallel Hash  (cost=28029.47..28029.47 rows=139793 width=32) (actual time=258.592..258.593 rows=234155 loops=3)
                                                              Buckets: 131072 (originally 131072)  Batches: 8 (originally 4)  Memory Usage: 6592kB
                                                              Buffers: shared hit=17074, temp read=2478 written=6440
                                                              I/O Timings: temp read=6.318 write=97.179
                                                              ->  Parallel Seq Scan on rfv_num v_7  (cost=0.00..28029.47 rows=139793 width=32) (actual time=46.479..123.180 rows=234155 loops=3)
                                                                    Filter: ((tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = 'a1442803-0ebd-40e0-b4b3-fcc2600fbfa3'::uuid) AND (value = '1'::double precision))
                                                                    Rows Removed by Filter: 266667
                                                                    Buffers: shared hit=17074
                                                  ->  Parallel Hash  (cost=28029.47..28029.47 rows=155595 width=32) (actual time=134.042..134.042 rows=103957 loops=3)
                                                        Buckets: 131072  Batches: 4  Memory Usage: 5952kB
                                                        Buffers: shared hit=17074, temp written=1504
                                                        I/O Timings: temp write=15.249
                                                        ->  Parallel Seq Scan on rfv_num v_6  (cost=0.00..28029.47 rows=155595 width=32) (actual time=21.808..96.163 rows=103957 loops=3)
                                                              Filter: ((value < '5'::double precision) AND (tenant_id = 'dcc69b4f-51af-49c0-93a9-425755f6a4cd'::uuid) AND (field_id = '9296a5d4-b4c8-4be2-b1f5-200ee75d84ca'::uuid))
                                                              Rows Removed by Filter: 396864
                                                              Buffers: shared hit=17074
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off', enable_nestloop = 'off'
Planning Time: 3.594 ms
Execution Time: 2205.984 ms
```
