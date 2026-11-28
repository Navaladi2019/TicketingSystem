# TICK-1009 S1/S2 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app, one run per cell)

### S1 num lt 5%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22531.01..24439.24 rows=1 width=1450) (actual time=53.305..53.612 rows=51 loops=1)
  Buffers: shared hit=22646
  ->  Nested Loop  (cost=22530.71..24430.91 rows=1 width=1415) (actual time=53.287..53.466 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22493
        ->  Subquery Scan on x  (cost=22530.43..24422.59 rows=1 width=1408) (actual time=53.275..53.353 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22340
              ->  Limit  (cost=22530.43..24421.45 rows=51 width=1408) (actual time=53.272..53.316 rows=51 loops=1)
                    Buffers: shared hit=22340
                    ->  Merge Append  (cost=22530.43..24643.92 rows=57 width=1408) (actual time=53.271..53.309 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22340
                          ->  Limit  (cost=22483.30..22483.43 rows=51 width=1408) (actual time=52.705..52.723 rows=51 loops=1)
                                Buffers: shared hit=22045
                                ->  Sort  (cost=22483.30..22483.67 rows=148 width=1408) (actual time=52.703..52.716 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=22045
                                      ->  Result  (cost=2127.02..22478.37 rows=148 width=1408) (actual time=16.886..51.738 rows=1186 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=22045
                                            ->  Hash Semi Join  (cost=2127.02..22478.37 rows=148 width=1400) (actual time=16.880..51.384 rows=1186 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=22045
                                                  ->  Bitmap Heap Scan on records r  (cost=477.88..20811.80 rows=6014 width=1400) (actual time=9.136..38.391 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=4.467..4.467 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=1403.16..1403.16 rows=19677 width=32) (actual time=7.694..7.694 rows=20292 loops=1)
                                                        Buckets: 32768  Batches: 1  Memory Usage: 1525kB
                                                        Buffers: shared hit=248
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..1403.16 rows=19677 width=32) (actual time=0.021..4.112 rows=20292 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '500'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=248
                          ->  Limit  (cost=0.86..1956.18 rows=4 width=1408) (actual time=0.392..0.393 rows=1 loops=1)
                                Buffers: shared hit=235
                                ->  Result  (cost=0.86..1956.18 rows=4 width=1408) (actual time=0.390..0.391 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=235
                                      ->  Nested Loop Semi Join  (cost=0.86..1956.18 rows=4 width=1400) (actual time=0.380..0.381 rows=1 loops=1)
                                            Buffers: shared hit=235
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=1400) (actual time=0.044..0.129 rows=46 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=51
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.40 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=46)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_1.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '500'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=184
                          ->  Limit  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.056..0.057 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.055..0.056 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=23
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=1400) (actual time=0.050..0.051 rows=1 loops=1)
                                            Buffers: shared hit=23
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.020..0.026 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=7
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '500'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=16
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.113..0.116 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.113..0.115 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1408) (actual time=0.090..0.106 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1400) (actual time=0.088..0.103 rows=1 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.063..0.080 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.040..0.043 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.034 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.011..0.011 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                        Filter: (value < '500'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.456 ms
Execution Time: 53.809 ms
```

### S1 num lt 5%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=4368.44..4368.45 rows=1 width=8) (actual time=16.053..16.061 rows=1 loops=1)
  Buffers: shared hit=13569
  ->  Limit  (cost=1649.68..4366.51 rows=154 width=4) (actual time=7.051..15.949 rows=1001 loops=1)
        Buffers: shared hit=13569
        ->  Result  (cost=1649.68..4366.51 rows=154 width=4) (actual time=7.050..15.828 rows=1001 loops=1)
              Buffers: shared hit=13569
              ->  Append  (cost=1649.68..4364.97 rows=154 width=0) (actual time=7.050..15.675 rows=1001 loops=1)
                    Buffers: shared hit=13569
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1649.68..2204.57 rows=148 width=0) (actual time=7.048..15.545 rows=1001 loops=1)
                          Buffers: shared hit=13569
                          ->  Limit  (cost=1649.68..2203.09 rows=148 width=4) (actual time=7.047..15.423 rows=1001 loops=1)
                                Buffers: shared hit=13569
                                ->  Result  (cost=1649.68..2203.09 rows=148 width=4) (actual time=7.046..15.320 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=13569
                                      ->  Hash Semi Join  (cost=1649.68..2203.09 rows=148 width=0) (actual time=7.040..15.178 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=13569
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.021..4.950 rows=20406 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13321
                                            ->  Hash  (cost=1403.16..1403.16 rows=19677 width=32) (actual time=6.992..6.993 rows=20292 loops=1)
                                                  Buckets: 32768  Batches: 1  Memory Usage: 1525kB
                                                  Buffers: shared hit=248
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..1403.16 rows=19677 width=32) (actual time=0.015..3.774 rows=20292 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '500'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=248
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1956.22 rows=4 width=0) (never executed)
                          ->  Limit  (cost=0.86..1956.18 rows=4 width=4) (never executed)
                                ->  Result  (cost=0.86..1956.18 rows=4 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1956.18 rows=4 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.40 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_1.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '500'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.06 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.86..158.05 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.86..158.05 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '500'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '500'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.676 ms
Execution Time: 16.184 ms
```

### S1 num lt 5%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=2268.41..2268.54 rows=51 width=1442) (actual time=151.774..151.792 rows=51 loops=1)
  Buffers: shared hit=81473
  ->  Sort  (cost=2268.41..2280.71 rows=4919 width=1442) (actual time=151.773..151.786 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 193kB
        Buffers: shared hit=81473
        ->  Result  (cost=1646.94..2104.31 rows=4919 width=1442) (actual time=10.021..137.654 rows=20292 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=81473
              ->  Hash Left Join  (cost=1646.94..2067.41 rows=4919 width=1415) (actual time=10.007..125.814 rows=20292 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=81473
                    ->  Nested Loop  (cost=1502.24..1909.80 rows=4919 width=1407) (actual time=9.862..117.463 rows=20292 loops=1)
                          Buffers: shared hit=81419
                          ->  Nested Loop  (cost=1501.82..1903.38 rows=1 width=71) (actual time=9.833..19.056 rows=20292 loops=1)
                                Buffers: shared hit=251
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.019..0.022 rows=1 loops=1)
                                      Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Filter: (id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=1501.54..1698.31 rows=19677 width=32) (actual time=9.812..15.553 rows=20292 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 2321kB
                                      Buffers: shared hit=248
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..1403.16 rows=19677 width=32) (actual time=0.018..3.452 rows=20292 loops=1)
                                            Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '500'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=248
                          ->  Index Scan using records_pkey on records r  (cost=0.42..5.51 rows=1 width=1400) (actual time=0.004..0.004 rows=1 loops=20292)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                Buffers: shared hit=81168
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.133..0.135 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.033..0.122 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.023..0.103 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.011..0.011 rows=53 loops=1)
                                            Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.805 ms
Execution Time: 151.949 ms
```

### S1 num lt 5%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=4602.95..4602.96 rows=1 width=8) (actual time=14.148..14.151 rows=1 loops=1)
  Buffers: shared hit=13130
  ->  Limit  (cost=1649.56..4590.43 rows=1001 width=4) (actual time=6.602..14.069 rows=1001 loops=1)
        Buffers: shared hit=13130
        ->  Result  (cost=1649.56..16101.28 rows=4919 width=4) (actual time=6.601..13.976 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=13130
              ->  Hash Semi Join  (cost=1649.56..16101.28 rows=4919 width=0) (actual time=6.596..13.851 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=13130
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13872.42 rows=200000 width=32) (actual time=0.015..4.275 rows=19835 loops=1)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=12882
                    ->  Hash  (cost=1403.16..1403.16 rows=19677 width=32) (actual time=6.550..6.551 rows=20292 loops=1)
                          Buckets: 32768  Batches: 1  Memory Usage: 1525kB
                          Buffers: shared hit=248
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..1403.16 rows=19677 width=32) (actual time=0.012..3.492 rows=20292 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '500'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=248
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.388 ms
Execution Time: 14.192 ms
```

### S1 num lt 10%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=24195.97..25980.25 rows=1 width=1450) (actual time=60.523..61.048 rows=51 loops=1)
  Buffers: shared hit=22851
  ->  Nested Loop  (cost=24195.67..25971.91 rows=1 width=1415) (actual time=60.508..60.905 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22698
        ->  Subquery Scan on x  (cost=24195.39..25963.60 rows=1 width=1408) (actual time=60.493..60.793 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22545
              ->  Limit  (cost=24195.39..25962.45 rows=51 width=1408) (actual time=60.490..60.756 rows=51 loops=1)
                    Buffers: shared hit=22545
                    ->  Merge Append  (cost=24195.39..26308.94 rows=61 width=1408) (actual time=60.489..60.749 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22545
                          ->  Limit  (cost=24148.26..24148.39 rows=51 width=1408) (actual time=60.191..60.207 rows=50 loops=1)
                                Buffers: shared hit=22280
                                ->  Sort  (cost=24148.26..24149.00 rows=297 width=1408) (actual time=60.190..60.201 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=22280
                                      ->  Result  (cost=3785.35..24138.35 rows=297 width=1408) (actual time=24.354..58.461 rows=2401 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=22280
                                            ->  Hash Semi Join  (cost=3785.35..24138.35 rows=297 width=1400) (actual time=24.350..57.828 rows=2401 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=22280
                                                  ->  Bitmap Heap Scan on records r  (cost=477.88..20811.80 rows=6014 width=1400) (actual time=9.079..36.579 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=4.530..4.531 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=2813.51..2813.51 rows=39515 width=32) (actual time=15.214..15.215 rows=40108 loops=1)
                                                        Buckets: 65536  Batches: 1  Memory Usage: 3019kB
                                                        Buffers: shared hit=483
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2813.51 rows=39515 width=32) (actual time=0.022..7.681 rows=40108 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '1000'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=483
                          ->  Limit  (cost=0.86..1956.18 rows=8 width=1408) (actual time=0.133..0.357 rows=2 loops=1)
                                Buffers: shared hit=205
                                ->  Result  (cost=0.86..1956.18 rows=8 width=1408) (actual time=0.131..0.354 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=205
                                      ->  Nested Loop Semi Join  (cost=0.86..1956.18 rows=8 width=1400) (actual time=0.120..0.342 rows=2 loops=1)
                                            Buffers: shared hit=205
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=1400) (actual time=0.021..0.095 rows=40 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=45
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.40 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=40)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_1.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '1000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=160
                          ->  Limit  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.052..0.053 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.052..0.053 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=23
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=1400) (actual time=0.048..0.049 rows=1 loops=1)
                                            Buffers: shared hit=23
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.018..0.022 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=7
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '1000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=16
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.107..0.110 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.106..0.109 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1408) (actual time=0.072..0.098 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1400) (actual time=0.069..0.095 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.057..0.072 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.046 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.038 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                        Filter: (value < '1000'::double precision)
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.339 ms
Execution Time: 61.248 ms
```

### S1 num lt 10%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=6034.16..6034.17 rows=1 width=8) (actual time=18.257..18.266 rows=1 loops=1)
  Buffers: shared hit=7028
  ->  Limit  (cost=3308.01..6030.33 rows=307 width=4) (actual time=13.422..18.152 rows=1001 loops=1)
        Buffers: shared hit=7028
        ->  Result  (cost=3308.01..6030.33 rows=307 width=4) (actual time=13.420..18.045 rows=1001 loops=1)
              Buffers: shared hit=7028
              ->  Append  (cost=3308.01..6027.26 rows=307 width=0) (actual time=13.420..17.907 rows=1001 loops=1)
                    Buffers: shared hit=7028
                    ->  Subquery Scan on "*SELECT* 1"  (cost=3308.01..3866.04 rows=297 width=0) (actual time=13.419..17.791 rows=1001 loops=1)
                          Buffers: shared hit=7028
                          ->  Limit  (cost=3308.01..3863.07 rows=297 width=4) (actual time=13.417..17.683 rows=1001 loops=1)
                                Buffers: shared hit=7028
                                ->  Result  (cost=3308.01..3863.07 rows=297 width=4) (actual time=13.416..17.593 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=7028
                                      ->  Hash Semi Join  (cost=3308.01..3863.07 rows=297 width=0) (actual time=13.411..17.471 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=7028
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.018..2.324 rows=9991 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=6545
                                            ->  Hash  (cost=2813.51..2813.51 rows=39515 width=32) (actual time=13.349..13.350 rows=40108 loops=1)
                                                  Buckets: 65536  Batches: 1  Memory Usage: 3019kB
                                                  Buffers: shared hit=483
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2813.51 rows=39515 width=32) (actual time=0.015..7.042 rows=40108 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '1000'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=483
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1956.26 rows=8 width=0) (never executed)
                          ->  Limit  (cost=0.86..1956.18 rows=8 width=4) (never executed)
                                ->  Result  (cost=0.86..1956.18 rows=8 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1956.18 rows=8 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.40 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_1.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '1000'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.06 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.86..158.05 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.86..158.05 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '1000'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '1000'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.578 ms
Execution Time: 18.385 ms
```

### S1 num lt 10%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.42..3469.93 rows=51 width=1442) (actual time=0.099..3.570 rows=51 loops=1)
  Buffers: shared hit=2395
  ->  Result  (cost=1.42..671872.44 rows=9879 width=1442) (actual time=0.098..3.557 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
        Buffers: shared hit=2395
        ->  Nested Loop Left Join  (cost=1.42..671798.35 rows=9879 width=1415) (actual time=0.086..3.502 rows=51 loops=1)
              Buffers: shared hit=2395
              ->  Nested Loop  (cost=1.12..668563.96 rows=9879 width=1407) (actual time=0.071..3.209 rows=51 loops=1)
                    Buffers: shared hit=2242
                    ->  Nested Loop Semi Join  (cost=0.85..668432.17 rows=9879 width=1400) (actual time=0.057..3.155 rows=51 loops=1)
                          Buffers: shared hit=2239
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..502921.38 rows=200000 width=1400) (actual time=0.025..0.968 rows=446 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                                Buffers: shared hit=455
                          ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..0.83 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=446)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                Filter: (value < '1000'::double precision)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=1784
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.009..0.010 rows=1 loops=1)
                                Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                Filter: (id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.928 ms
Execution Time: 3.646 ms
```

### S1 num lt 10%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=4790.33..4790.34 rows=1 width=8) (actual time=23.050..23.073 rows=1 loops=1)
  Buffers: shared hit=6836
  ->  Limit  (cost=3307.89..4777.81 rows=1001 width=4) (actual time=17.653..22.956 rows=1001 loops=1)
        Buffers: shared hit=6836
        ->  Result  (cost=3307.89..17814.79 rows=9879 width=4) (actual time=17.652..22.842 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=6836
              ->  Hash Semi Join  (cost=3307.89..17814.79 rows=9879 width=0) (actual time=17.645..22.680 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=6836
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13872.42 rows=200000 width=32) (actual time=0.018..2.768 rows=9799 loops=1)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=6353
                    ->  Hash  (cost=2813.51..2813.51 rows=39515 width=32) (actual time=17.575..17.577 rows=40108 loops=1)
                          Buckets: 65536  Batches: 1  Memory Usage: 3019kB
                          Buffers: shared hit=483
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2813.51 rows=39515 width=32) (actual time=0.018..9.233 rows=40108 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '1000'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=483
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.493 ms
Execution Time: 23.135 ms
```

### S1 num lt 20%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=21931.28..26127.27 rows=1 width=1450) (actual time=65.049..79.922 rows=51 loops=1)
  Buffers: shared hit=23121, temp read=1244 written=4112
  I/O Timings: temp read=2.960 write=25.451
  ->  Nested Loop  (cost=21930.99..26118.93 rows=1 width=1415) (actual time=65.029..79.713 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22968, temp read=1244 written=4112
        I/O Timings: temp read=2.960 write=25.451
        ->  Subquery Scan on x  (cost=21930.70..26110.62 rows=1 width=1408) (actual time=65.011..79.557 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22815, temp read=1244 written=4112
              I/O Timings: temp read=2.960 write=25.451
              ->  Limit  (cost=21930.70..26109.47 rows=51 width=1408) (actual time=65.008..79.498 rows=51 loops=1)
                    Buffers: shared hit=22815, temp read=1244 written=4112
                    I/O Timings: temp read=2.960 write=25.451
                    ->  Merge Append  (cost=21930.70..27502.40 rows=68 width=1408) (actual time=65.006..79.486 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22815, temp read=1244 written=4112
                          I/O Timings: temp read=2.960 write=25.451
                          ->  Limit  (cost=21883.58..25341.74 rows=51 width=1408) (actual time=64.748..79.160 rows=50 loops=1)
                                Buffers: shared hit=22717, temp read=1244 written=4112
                                I/O Timings: temp read=2.960 write=25.451
                                ->  Result  (cost=21883.58..61822.02 rows=589 width=1408) (actual time=64.747..79.149 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=22717, temp read=1244 written=4112
                                      I/O Timings: temp read=2.960 write=25.451
                                      ->  Nested Loop Semi Join  (cost=21883.58..61822.02 rows=589 width=1400) (actual time=64.742..79.123 rows=50 loops=1)
                                            Buffers: shared hit=22717, temp read=1244 written=4112
                                            I/O Timings: temp read=2.960 write=25.451
                                            ->  Gather Merge  (cost=21883.14..22583.57 rows=6014 width=1400) (actual time=64.674..77.970 rows=222 loops=1)
                                                  Workers Planned: 2
                                                  Workers Launched: 2
                                                  Buffers: shared hit=21829, temp read=1244 written=4112
                                                  I/O Timings: temp read=2.960 write=25.451
                                                  ->  Sort  (cost=20883.11..20889.38 rows=2506 width=1400) (actual time=59.709..59.777 rows=103 loops=3)
                                                        Sort Key: r.updated_at DESC, r.id DESC
                                                        Sort Method: external merge  Disk: 16208kB
                                                        Buffers: shared hit=21829, temp read=1244 written=4112
                                                        I/O Timings: temp read=2.960 write=25.451
                                                        Worker 0:  Sort Method: external merge  Disk: 8520kB
                                                        Worker 1:  Sort Method: external merge  Disk: 8104kB
                                                        ->  Parallel Bitmap Heap Scan on records r  (cost=477.88..20741.64 rows=2506 width=1400) (actual time=8.239..37.601 rows=8039 loops=3)
                                                              Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Heap Blocks: exact=10584
                                                              Buffers: shared hit=21797
                                                              ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=6.558..6.558 rows=24118 loops=1)
                                                                    Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=375
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..6.51 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=222)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '2000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=888
                          ->  Limit  (cost=0.86..1956.18 rows=15 width=1408) (actual time=0.072..0.113 rows=2 loops=1)
                                Buffers: shared hit=38
                                ->  Result  (cost=0.86..1956.18 rows=15 width=1408) (actual time=0.070..0.110 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=38
                                      ->  Nested Loop Semi Join  (cost=0.86..1956.18 rows=15 width=1400) (actual time=0.059..0.097 rows=2 loops=1)
                                            Buffers: shared hit=38
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=1400) (actual time=0.021..0.038 rows=7 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=10
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.40 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=7)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_1.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '2000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=28
                          ->  Limit  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.053..0.056 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.053..0.055 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=23
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=1400) (actual time=0.050..0.052 rows=1 loops=1)
                                            Buffers: shared hit=23
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.015..0.023 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=7
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '2000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=16
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.130..0.137 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.130..0.136 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1408) (actual time=0.084..0.120 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1400) (actual time=0.081..0.115 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.060..0.080 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.046 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.037 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.023 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.009..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                        Filter: (value < '2000'::double precision)
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.704 ms
Execution Time: 83.104 ms
```

### S1 num lt 20%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9298.13..9298.14 rows=1 width=8) (actual time=36.260..36.271 rows=1 loops=1)
  Buffers: shared hit=4283
  ->  Limit  (cost=6557.51..9290.55 rows=606 width=4) (actual time=32.740..36.134 rows=1001 loops=1)
        Buffers: shared hit=4283
        ->  Result  (cost=6557.51..9290.55 rows=606 width=4) (actual time=32.739..36.015 rows=1001 loops=1)
              Buffers: shared hit=4283
              ->  Append  (cost=6557.51..9284.49 rows=606 width=0) (actual time=32.738..35.856 rows=1001 loops=1)
                    Buffers: shared hit=4283
                    ->  Subquery Scan on "*SELECT* 1"  (cost=6557.51..7121.71 rows=589 width=0) (actual time=32.737..35.725 rows=1001 loops=1)
                          Buffers: shared hit=4283
                          ->  Limit  (cost=6557.51..7115.82 rows=589 width=4) (actual time=32.736..35.607 rows=1001 loops=1)
                                Buffers: shared hit=4283
                                ->  Result  (cost=6557.51..7115.82 rows=589 width=4) (actual time=32.735..35.512 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=4283
                                      ->  Hash Semi Join  (cost=6557.51..7115.82 rows=589 width=0) (actual time=32.727..35.371 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=4283
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.024..1.333 rows=5128 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=3328
                                            ->  Hash  (cost=5576.76..5576.76 rows=78415 width=32) (actual time=32.604..32.605 rows=79979 loops=1)
                                                  Buckets: 131072  Batches: 1  Memory Usage: 6023kB
                                                  Buffers: shared hit=955
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5576.76 rows=78415 width=32) (actual time=0.022..15.755 rows=79979 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '2000'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=955
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1956.33 rows=15 width=0) (never executed)
                          ->  Limit  (cost=0.86..1956.18 rows=15 width=4) (never executed)
                                ->  Result  (cost=0.86..1956.18 rows=15 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1956.18 rows=15 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.40 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_1.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '2000'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.06 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.86..158.05 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.86..158.05 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '2000'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '2000'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.894 ms
Execution Time: 36.428 ms
```

### S1 num lt 20%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.42..1758.34 rows=51 width=1442) (actual time=0.099..2.053 rows=51 loops=1)
  Buffers: shared hit=1342
  ->  Result  (cost=1.42..675348.12 rows=19604 width=1442) (actual time=0.097..2.040 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
        Buffers: shared hit=1342
        ->  Nested Loop Left Join  (cost=1.42..675201.09 rows=19604 width=1415) (actual time=0.085..1.990 rows=51 loops=1)
              Buffers: shared hit=1342
              ->  Nested Loop  (cost=1.12..668782.77 rows=19604 width=1407) (actual time=0.071..1.727 rows=51 loops=1)
                    Buffers: shared hit=1189
                    ->  Nested Loop Semi Join  (cost=0.85..668529.42 rows=19604 width=1400) (actual time=0.058..1.677 rows=51 loops=1)
                          Buffers: shared hit=1186
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..502921.38 rows=200000 width=1400) (actual time=0.026..0.533 rows=236 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                                Buffers: shared hit=242
                          ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..0.83 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=236)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                Filter: (value < '2000'::double precision)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=944
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.008..0.009 rows=1 loops=1)
                                Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                Filter: (id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.982 ms
Execution Time: 2.129 ms
```

### S1 num lt 20%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=7316.16..7316.17 rows=1 width=8) (actual time=36.857..36.862 rows=1 loops=1)
  Buffers: shared hit=4190
  ->  Limit  (cost=6557.39..7303.65 rows=1001 width=4) (actual time=33.848..36.771 rows=1001 loops=1)
        Buffers: shared hit=4190
        ->  Result  (cost=6557.39..21172.48 rows=19604 width=4) (actual time=33.846..36.664 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=4190
              ->  Hash Semi Join  (cost=6557.39..21172.48 rows=19604 width=0) (actual time=33.841..36.510 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=4190
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13872.42 rows=200000 width=32) (actual time=0.019..1.324 rows=4983 loops=1)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=3235
                    ->  Hash  (cost=5576.76..5576.76 rows=78415 width=32) (actual time=33.731..33.732 rows=79979 loops=1)
                          Buckets: 131072  Batches: 1  Memory Usage: 6023kB
                          Buffers: shared hit=955
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5576.76 rows=78415 width=32) (actual time=0.019..17.170 rows=79979 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '2000'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=955
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.447 ms
Execution Time: 36.931 ms
```

### S1 text eq 5%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22455.92..23792.45 rows=1 width=1450) (actual time=50.689..51.302 rows=51 loops=1)
  Buffers: shared hit=22656
  ->  Nested Loop  (cost=22455.62..23784.11 rows=1 width=1415) (actual time=50.675..51.154 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22503
        ->  Subquery Scan on x  (cost=22455.34..23775.80 rows=1 width=1408) (actual time=50.664..51.042 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22350
              ->  Limit  (cost=22455.34..23774.65 rows=51 width=1408) (actual time=50.662..51.005 rows=51 loops=1)
                    Buffers: shared hit=22350
                    ->  Merge Append  (cost=22455.34..23929.86 rows=57 width=1408) (actual time=50.660..50.998 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22350
                          ->  Limit  (cost=22407.81..22407.94 rows=51 width=1408) (actual time=50.446..50.464 rows=49 loops=1)
                                Buffers: shared hit=22052
                                ->  Sort  (cost=22407.81..22408.15 rows=138 width=1408) (actual time=50.444..50.457 rows=49 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 207kB
                                      Buffers: shared hit=22052
                                      ->  Result  (cost=2051.79..22403.20 rows=138 width=1408) (actual time=15.970..49.512 rows=1249 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=22052
                                            ->  Hash Semi Join  (cost=2051.79..22403.20 rows=138 width=1400) (actual time=15.965..49.153 rows=1249 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=22052
                                                  ->  Bitmap Heap Scan on records r  (cost=477.88..20811.80 rows=6014 width=1400) (actual time=8.375..36.599 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=3.863..3.863 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=1340.56..1340.56 rows=18667 width=32) (actual time=7.559..7.560 rows=20087 loops=1)
                                                        Buckets: 32768  Batches: 1  Memory Usage: 1512kB
                                                        Buffers: shared hit=255
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..1340.56 rows=18667 width=32) (actual time=0.030..4.118 rows=20087 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=255
                          ->  Limit  (cost=0.99..1363.94 rows=4 width=1408) (actual time=0.073..0.300 rows=2 loops=1)
                                Buffers: shared hit=181
                                ->  Result  (cost=0.99..1363.94 rows=4 width=1408) (actual time=0.072..0.298 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=181
                                      ->  Nested Loop Semi Join  (cost=0.99..1363.94 rows=4 width=1400) (actual time=0.060..0.284 rows=2 loops=1)
                                            Buffers: shared hit=181
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=1400) (actual time=0.023..0.116 rows=35 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=40
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=35)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=141
                          ->  Limit  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.047..0.119 rows=2 loops=1)
                                Buffers: shared hit=80
                                ->  Result  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.046..0.118 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=80
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=1400) (actual time=0.044..0.115 rows=2 loops=1)
                                            Buffers: shared hit=80
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.013..0.043 rows=15 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=19
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=15)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=61
                          ->  Limit  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.090..0.092 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.089..0.092 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1408) (actual time=0.083..0.085 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1400) (actual time=0.080..0.082 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.049..0.064 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.037..0.039 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.025..0.030 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.652 ms
Execution Time: 51.498 ms
```

### S1 text eq 5%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=3654.34..3654.35 rows=1 width=8) (actual time=15.317..15.326 rows=1 loops=1)
  Buffers: shared hit=12922
  ->  Limit  (cost=1574.46..3652.54 rows=144 width=4) (actual time=7.066..15.207 rows=1001 loops=1)
        Buffers: shared hit=12922
        ->  Result  (cost=1574.46..3652.54 rows=144 width=4) (actual time=7.065..15.085 rows=1001 loops=1)
              Buffers: shared hit=12922
              ->  Append  (cost=1574.46..3651.10 rows=144 width=0) (actual time=7.064..14.938 rows=1001 loops=1)
                    Buffers: shared hit=12922
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1574.46..2129.30 rows=138 width=0) (actual time=7.063..14.813 rows=1001 loops=1)
                          Buffers: shared hit=12922
                          ->  Limit  (cost=1574.46..2127.92 rows=138 width=4) (actual time=7.062..14.692 rows=1001 loops=1)
                                Buffers: shared hit=12922
                                ->  Result  (cost=1574.46..2127.92 rows=138 width=4) (actual time=7.062..14.596 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=12922
                                      ->  Hash Semi Join  (cost=1574.46..2127.92 rows=138 width=0) (actual time=7.056..14.462 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=12922
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.020..4.493 rows=19401 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12668
                                            ->  Hash  (cost=1340.56..1340.56 rows=18667 width=32) (actual time=7.007..7.007 rows=20087 loops=1)
                                                  Buckets: 32768  Batches: 1  Memory Usage: 1512kB
                                                  Buffers: shared hit=254
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..1340.56 rows=18667 width=32) (actual time=0.018..3.762 rows=20087 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=254
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..1363.98 rows=4 width=0) (never executed)
                          ->  Limit  (cost=0.99..1363.94 rows=4 width=4) (never executed)
                                ->  Result  (cost=0.99..1363.94 rows=4 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..1363.94 rows=4 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.99..111.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.99..111.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.756 ms
Execution Time: 15.446 ms
```

### S1 text eq 5%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=2164.55..2164.68 rows=51 width=1442) (actual time=118.993..119.008 rows=51 loops=1)
  Buffers: shared hit=80660
  ->  Sort  (cost=2164.55..2176.05 rows=4598 width=1442) (actual time=118.992..119.003 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 201kB
        Buffers: shared hit=80660
        ->  Result  (cost=1579.30..2011.15 rows=4598 width=1442) (actual time=8.079..107.435 rows=20087 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=80660
              ->  Hash Left Join  (cost=1579.30..1976.67 rows=4598 width=1415) (actual time=8.070..98.050 rows=20087 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=80660
                    ->  Nested Loop  (cost=1434.59..1819.89 rows=4598 width=1407) (actual time=7.940..91.411 rows=20087 loops=1)
                          Buffers: shared hit=80606
                          ->  Nested Loop  (cost=1434.17..1813.41 rows=1 width=71) (actual time=7.925..14.741 rows=20087 loops=1)
                                Buffers: shared hit=258
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.013..0.015 rows=1 loops=1)
                                      Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Filter: (id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=1433.90..1619.51 rows=18561 width=32) (actual time=7.910..12.091 rows=20087 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 2321kB
                                      Buffers: shared hit=255
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..1340.56 rows=18667 width=32) (actual time=0.020..3.036 rows=20087 loops=1)
                                            Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=255
                          ->  Index Scan using records_pkey on records r  (cost=0.42..5.60 rows=1 width=1400) (actual time=0.003..0.003 rows=1 loops=20087)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                Buffers: shared hit=80348
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.123..0.124 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.038..0.114 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.034..0.102 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.024..0.024 rows=53 loops=1)
                                            Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.732 ms
Execution Time: 119.143 ms
```

### S1 text eq 5%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=4733.51..4733.52 rows=1 width=8) (actual time=11.419..11.422 rows=1 loops=1)
  Buffers: shared hit=12381
  ->  Limit  (cost=1574.34..4721.00 rows=1001 width=4) (actual time=5.651..11.356 rows=1001 loops=1)
        Buffers: shared hit=12381
        ->  Result  (cost=1574.34..16028.24 rows=4598 width=4) (actual time=5.650..11.280 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=12381
              ->  Hash Semi Join  (cost=1574.34..16028.24 rows=4598 width=0) (actual time=5.646..11.177 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=12381
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13872.42 rows=200000 width=32) (actual time=0.014..3.297 rows=18680 loops=1)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=12127
                    ->  Hash  (cost=1340.56..1340.56 rows=18667 width=32) (actual time=5.610..5.611 rows=20087 loops=1)
                          Buckets: 32768  Batches: 1  Memory Usage: 1512kB
                          Buffers: shared hit=254
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..1340.56 rows=18667 width=32) (actual time=0.015..3.038 rows=20087 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v5'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=254
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.342 ms
Execution Time: 11.456 ms
```

### S1 text eq 10%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=24144.80..25415.40 rows=1 width=1450) (actual time=75.260..75.684 rows=51 loops=1)
  Buffers: shared hit=22857
  ->  Nested Loop  (cost=24144.50..25407.06 rows=1 width=1415) (actual time=75.240..75.510 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22704
        ->  Subquery Scan on x  (cost=24144.22..25398.75 rows=1 width=1408) (actual time=75.228..75.382 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22551
              ->  Limit  (cost=24144.22..25397.60 rows=51 width=1408) (actual time=75.224..75.339 rows=51 loops=1)
                    Buffers: shared hit=22551
                    ->  Merge Append  (cost=24144.22..25618.79 rows=60 width=1408) (actual time=75.223..75.330 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22551
                          ->  Limit  (cost=24096.69..24096.81 rows=51 width=1408) (actual time=74.474..74.499 rows=49 loops=1)
                                Buffers: shared hit=22295
                                ->  Sort  (cost=24096.69..24097.40 rows=286 width=1408) (actual time=74.472..74.492 rows=49 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 191kB
                                      Buffers: shared hit=22295
                                      ->  Result  (cost=3733.90..24087.15 rows=286 width=1408) (actual time=27.037..72.571 rows=2355 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=22295
                                            ->  Hash Semi Join  (cost=3733.90..24087.15 rows=286 width=1400) (actual time=27.029..71.839 rows=2355 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=22295
                                                  ->  Bitmap Heap Scan on records r  (cost=477.88..20811.80 rows=6014 width=1400) (actual time=9.633..44.966 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=4.425..4.425 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=2773.34..2773.34 rows=38613 width=32) (actual time=17.233..17.234 rows=39642 loops=1)
                                                        Buckets: 65536  Batches: 1  Memory Usage: 2990kB
                                                        Buffers: shared hit=498
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..2773.34 rows=38613 width=32) (actual time=0.056..9.184 rows=39642 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=498
                          ->  Limit  (cost=0.99..1363.94 rows=7 width=1408) (actual time=0.259..0.315 rows=3 loops=1)
                                Buffers: shared hit=69
                                ->  Result  (cost=0.99..1363.94 rows=7 width=1408) (actual time=0.254..0.309 rows=3 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=69
                                      ->  Nested Loop Semi Join  (cost=0.99..1363.94 rows=7 width=1400) (actual time=0.229..0.283 rows=3 loops=1)
                                            Buffers: shared hit=69
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=1400) (actual time=0.086..0.128 rows=13 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=16
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=13)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=53
                          ->  Limit  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.334..0.335 rows=1 loops=1)
                                Buffers: shared hit=150
                                ->  Result  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.332..0.333 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=150
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=1400) (actual time=0.324..0.324 rows=1 loops=1)
                                            Buffers: shared hit=150
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.029..0.119 rows=29 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=33
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=29)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=117
                          ->  Limit  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.147..0.150 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.145..0.148 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1408) (actual time=0.129..0.131 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1400) (actual time=0.122..0.125 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.078..0.097 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.061..0.063 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.043..0.049 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.009..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 3.155 ms
Execution Time: 75.942 ms
```

### S1 text eq 10%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=5334.58..5334.59 rows=1 width=8) (actual time=24.958..24.969 rows=1 loops=1)
  Buffers: shared hit=15900
  ->  Limit  (cost=914.74..5330.89 rows=295 width=4) (actual time=16.045..24.845 rows=1001 loops=1)
        Buffers: shared hit=15900
        ->  Result  (cost=914.74..5330.89 rows=295 width=4) (actual time=16.044..24.715 rows=1001 loops=1)
              Buffers: shared hit=15900
              ->  Append  (cost=914.74..5327.94 rows=295 width=0) (actual time=16.043..24.561 rows=1001 loops=1)
                    Buffers: shared hit=15900
                    ->  Subquery Scan on "*SELECT* 1"  (cost=914.74..3805.36 rows=286 width=0) (actual time=16.041..24.437 rows=1001 loops=1)
                          Buffers: shared hit=15900
                          ->  Limit  (cost=914.74..3802.50 rows=286 width=4) (actual time=16.039..24.303 rows=1001 loops=1)
                                Buffers: shared hit=15900
                                ->  Result  (cost=914.74..3802.50 rows=286 width=4) (actual time=16.038..24.191 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=15900
                                      ->  Merge Semi Join  (cost=914.74..3802.50 rows=286 width=0) (actual time=16.029..24.033 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15900
                                            ->  Sort  (cost=914.02..929.06 rows=6014 width=32) (actual time=15.976..17.541 rows=10165 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: quicksort  Memory: 2087kB
                                                  Buffers: shared hit=15681
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.029..8.072 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..2773.34 rows=38613 width=32) (actual time=0.031..3.202 rows=16798 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=219
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..1364.01 rows=7 width=0) (never executed)
                          ->  Limit  (cost=0.99..1363.94 rows=7 width=4) (never executed)
                                ->  Result  (cost=0.99..1363.94 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..1363.94 rows=7 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.99..111.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.99..111.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.174 ms
Execution Time: 25.127 ms
```

### S1 text eq 10%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.55..3758.96 rows=51 width=1442) (actual time=0.127..3.605 rows=51 loops=1)
  Buffers: shared hit=2681
  ->  Result  (cost=1.55..700648.56 rows=9510 width=1442) (actual time=0.127..3.595 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
        Buffers: shared hit=2681
        ->  Nested Loop Left Join  (cost=1.55..700577.24 rows=9510 width=1415) (actual time=0.117..3.550 rows=51 loops=1)
              Buffers: shared hit=2681
              ->  Nested Loop  (cost=1.25..697463.65 rows=9510 width=1407) (actual time=0.105..3.332 rows=51 loops=1)
                    Buffers: shared hit=2528
                    ->  Nested Loop Semi Join  (cost=0.98..697336.48 rows=9510 width=1400) (actual time=0.095..3.286 rows=51 loops=1)
                          Buffers: shared hit=2525
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..502921.38 rows=200000 width=1400) (actual time=0.020..0.941 rows=503 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                                Buffers: shared hit=512
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..0.97 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=503)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C") AND (record_id = r.id))
                                Heap Fetches: 0
                                Buffers: shared hit=2013
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.007..0.008 rows=1 loops=1)
                                Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                Filter: (id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.940 ms
Execution Time: 3.674 ms
```

### S1 text eq 10%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=4796.74..4796.75 rows=1 width=8) (actual time=21.659..21.663 rows=1 loops=1)
  Buffers: shared hit=7194
  ->  Limit  (cost=3256.44..4784.23 rows=1001 width=4) (actual time=16.402..21.570 rows=1001 loops=1)
        Buffers: shared hit=7194
        ->  Result  (cost=3256.44..17771.13 rows=9510 width=4) (actual time=16.400..21.460 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=7194
              ->  Hash Semi Join  (cost=3256.44..17771.13 rows=9510 width=0) (actual time=16.395..21.309 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=7194
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13872.42 rows=200000 width=32) (actual time=0.016..2.713 rows=10318 loops=1)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=6697
                    ->  Hash  (cost=2773.34..2773.34 rows=38613 width=32) (actual time=16.323..16.324 rows=39642 loops=1)
                          Buckets: 65536  Batches: 1  Memory Usage: 2990kB
                          Buffers: shared hit=497
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..2773.34 rows=38613 width=32) (actual time=0.020..8.762 rows=39642 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v10'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=497
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.467 ms
Execution Time: 21.713 ms
```

### S1 text eq 20%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=21931.81..24666.14 rows=1 width=1450) (actual time=59.961..73.307 rows=51 loops=1)
  Buffers: shared hit=22998, temp read=1245 written=4113
  I/O Timings: temp read=2.958 write=23.650
  ->  Nested Loop  (cost=21931.51..24657.81 rows=1 width=1415) (actual time=59.940..73.127 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22845, temp read=1245 written=4113
        I/O Timings: temp read=2.958 write=23.650
        ->  Subquery Scan on x  (cost=21931.23..24649.49 rows=1 width=1408) (actual time=59.923..72.983 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22692, temp read=1245 written=4113
              I/O Timings: temp read=2.958 write=23.650
              ->  Limit  (cost=21931.23..24648.35 rows=51 width=1408) (actual time=59.921..72.932 rows=51 loops=1)
                    Buffers: shared hit=22692, temp read=1245 written=4113
                    I/O Timings: temp read=2.958 write=23.650
                    ->  Merge Append  (cost=21931.23..25554.05 rows=68 width=1408) (actual time=59.919..72.921 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22692, temp read=1245 written=4113
                          I/O Timings: temp read=2.958 write=23.650
                          ->  Limit  (cost=21883.70..24031.96 rows=51 width=1408) (actual time=59.657..72.629 rows=51 loops=1)
                                Buffers: shared hit=22606, temp read=1245 written=4113
                                I/O Timings: temp read=2.958 write=23.650
                                ->  Result  (cost=21883.70..46988.81 rows=596 width=1408) (actual time=59.656..72.619 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=22606, temp read=1245 written=4113
                                      I/O Timings: temp read=2.958 write=23.650
                                      ->  Nested Loop Semi Join  (cost=21883.70..46988.81 rows=596 width=1400) (actual time=59.651..72.597 rows=51 loops=1)
                                            Buffers: shared hit=22606, temp read=1245 written=4113
                                            I/O Timings: temp read=2.958 write=23.650
                                            ->  Gather Merge  (cost=21883.14..22583.57 rows=6014 width=1400) (actual time=59.588..71.562 rows=194 loops=1)
                                                  Workers Planned: 2
                                                  Workers Launched: 2
                                                  Buffers: shared hit=21829, temp read=1245 written=4113
                                                  I/O Timings: temp read=2.958 write=23.650
                                                  ->  Sort  (cost=20883.11..20889.38 rows=2506 width=1400) (actual time=55.314..55.366 rows=96 loops=3)
                                                        Sort Key: r.updated_at DESC, r.id DESC
                                                        Sort Method: external merge  Disk: 16088kB
                                                        Buffers: shared hit=21829, temp read=1245 written=4113
                                                        I/O Timings: temp read=2.958 write=23.650
                                                        Worker 0:  Sort Method: external merge  Disk: 8728kB
                                                        Worker 1:  Sort Method: external merge  Disk: 8024kB
                                                        ->  Parallel Bitmap Heap Scan on records r  (cost=477.88..20741.64 rows=2506 width=1400) (actual time=7.899..34.504 rows=8039 loops=3)
                                                              Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Heap Blocks: exact=10483
                                                              Buffers: shared hit=21797
                                                              ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=6.027..6.027 rows=24118 loops=1)
                                                                    Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Buffers: shared hit=375
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..4.05 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=194)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=777
                          ->  Limit  (cost=0.99..1363.94 rows=15 width=1408) (actual time=0.084..0.087 rows=1 loops=1)
                                Buffers: shared hit=34
                                ->  Result  (cost=0.99..1363.94 rows=15 width=1408) (actual time=0.083..0.085 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=34
                                      ->  Nested Loop Semi Join  (cost=0.99..1363.94 rows=15 width=1400) (actual time=0.073..0.074 rows=1 loops=1)
                                            Buffers: shared hit=34
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=1400) (actual time=0.021..0.034 rows=6 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=9
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=6)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=25
                          ->  Limit  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.045..0.048 rows=1 loops=1)
                                Buffers: shared hit=14
                                ->  Result  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.045..0.047 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=14
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=1400) (actual time=0.041..0.043 rows=1 loops=1)
                                            Buffers: shared hit=14
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.015..0.017 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=9
                          ->  Limit  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.129..0.138 rows=1 loops=1)
                                Buffers: shared hit=38
                                ->  Sort  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.129..0.138 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=38
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1408) (actual time=0.111..0.119 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=38
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1400) (actual time=0.107..0.115 rows=1 loops=1)
                                                  Buffers: shared hit=38
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.067..0.089 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.049..0.054 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.045 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.029 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.010..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 3.132 ms
Execution Time: 76.485 ms
```

### S1 text eq 20%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=8439.43..8439.44 rows=1 width=8) (actual time=20.606..20.616 rows=1 loops=1)
  Buffers: shared hit=15888
  ->  Limit  (cost=914.74..8431.77 rows=613 width=4) (actual time=13.942..20.506 rows=1001 loops=1)
        Buffers: shared hit=15888
        ->  Result  (cost=914.74..8431.77 rows=613 width=4) (actual time=13.940..20.390 rows=1001 loops=1)
              Buffers: shared hit=15888
              ->  Append  (cost=914.74..8425.64 rows=613 width=0) (actual time=13.939..20.253 rows=1001 loops=1)
                    Buffers: shared hit=15888
                    ->  Subquery Scan on "*SELECT* 1"  (cost=914.74..6901.39 rows=596 width=0) (actual time=13.938..20.143 rows=1001 loops=1)
                          Buffers: shared hit=15888
                          ->  Limit  (cost=914.74..6895.43 rows=596 width=4) (actual time=13.937..20.025 rows=1001 loops=1)
                                Buffers: shared hit=15888
                                ->  Result  (cost=914.74..6895.43 rows=596 width=4) (actual time=13.935..19.925 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=15888
                                      ->  Merge Semi Join  (cost=914.74..6895.43 rows=596 width=0) (actual time=13.929..19.788 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15888
                                            ->  Sort  (cost=914.02..929.06 rows=6014 width=32) (actual time=13.892..14.585 rows=5038 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: quicksort  Memory: 2087kB
                                                  Buffers: shared hit=15681
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.022..6.646 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..5774.16 rows=80427 width=32) (actual time=0.027..2.897 rows=16994 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=207
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..1364.09 rows=15 width=0) (never executed)
                          ->  Limit  (cost=0.99..1363.94 rows=15 width=4) (never executed)
                                ->  Result  (cost=0.99..1363.94 rows=15 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..1363.94 rows=15 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..652.87 rows=155 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.99..111.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.99..111.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.806 ms
Execution Time: 20.771 ms
```

### S1 text eq 20%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.55..1814.90 rows=51 width=1442) (actual time=0.131..1.979 rows=51 loops=1)
  Buffers: shared hit=1258
  ->  Result  (cost=1.55..704329.39 rows=19809 width=1442) (actual time=0.130..1.969 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
        Buffers: shared hit=1258
        ->  Nested Loop Left Join  (cost=1.55..704180.82 rows=19809 width=1415) (actual time=0.119..1.922 rows=51 loops=1)
              Buffers: shared hit=1258
              ->  Nested Loop  (cost=1.25..697695.38 rows=19809 width=1407) (actual time=0.106..1.715 rows=51 loops=1)
                    Buffers: shared hit=1105
                    ->  Nested Loop Semi Join  (cost=0.98..697439.47 rows=19809 width=1400) (actual time=0.095..1.669 rows=51 loops=1)
                          Buffers: shared hit=1102
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..502921.38 rows=200000 width=1400) (actual time=0.022..0.481 rows=219 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                                Buffers: shared hit=225
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..0.97 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=219)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C") AND (record_id = r.id))
                                Heap Fetches: 0
                                Buffers: shared hit=877
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.008..0.008 rows=1 loops=1)
                                Index Cond: (tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                Filter: (id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 1.010 ms
Execution Time: 2.049 ms
```

### S1 text eq 20%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND (((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[]))))) OR true) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=7532.35..7532.36 rows=1 width=8) (actual time=37.408..37.411 rows=1 loops=1)
  Buffers: shared hit=4297
  ->  Limit  (cost=6779.94..7519.84 rows=1001 width=4) (actual time=34.496..37.325 rows=1001 loops=1)
        Buffers: shared hit=4297
        ->  Result  (cost=6779.94..21422.07 rows=19809 width=4) (actual time=34.494..37.207 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
              Buffers: shared hit=4297
              ->  Hash Semi Join  (cost=6779.94..21422.07 rows=19809 width=0) (actual time=34.488..37.065 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=4297
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13872.42 rows=200000 width=32) (actual time=0.020..1.309 rows=5087 loops=1)
                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=3311
                    ->  Hash  (cost=5774.16..5774.16 rows=80427 width=32) (actual time=34.369..34.370 rows=79871 loops=1)
                          Buckets: 131072  Batches: 1  Memory Usage: 6016kB
                          Buffers: shared hit=986
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..5774.16 rows=80427 width=32) (actual time=0.026..18.328 rows=79871 loops=1)
                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v20'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=986
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=1
Planning Time: 0.494 ms
Execution Time: 37.476 ms
```

### S2 num lt 2%, agents seeing 6% (control)

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22759.06..22925.62 rows=1 width=1450) (actual time=61.457..61.837 rows=51 loops=1)
  Buffers: shared hit=23047
  ->  Nested Loop  (cost=22758.76..22917.29 rows=1 width=1415) (actual time=61.437..61.663 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22894
        ->  Subquery Scan on x  (cost=22758.48..22908.97 rows=1 width=1408) (actual time=61.424..61.527 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22741
              ->  Limit  (cost=22758.48..22907.83 rows=51 width=1408) (actual time=61.421..61.484 rows=51 loops=1)
                    Buffers: shared hit=22741
                    ->  Merge Append  (cost=22758.48..22916.61 rows=54 width=1408) (actual time=61.419..61.476 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22741
                          ->  Limit  (cost=21449.17..21449.30 rows=51 width=1408) (actual time=56.077..56.105 rows=51 loops=1)
                                Buffers: shared hit=21898
                                ->  Sort  (cost=21449.17..21449.31 rows=56 width=1408) (actual time=56.075..56.097 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 207kB
                                      Buffers: shared hit=21898
                                      ->  Result  (cost=1097.22..21447.54 rows=56 width=1408) (actual time=13.781..55.515 rows=481 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=21898
                                            ->  Hash Semi Join  (cost=1097.22..21447.54 rows=56 width=1400) (actual time=13.775..55.296 rows=481 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21898
                                                  ->  Bitmap Heap Scan on records r  (cost=477.88..20811.80 rows=6014 width=1400) (actual time=10.020..44.834 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=4.554..4.555 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=3.502..3.503 rows=8098 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.036..1.949 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=1263.05..1263.05 rows=1 width=1408) (actual time=4.835..4.838 rows=1 loops=1)
                                Buffers: shared hit=622
                                ->  Sort  (cost=1263.05..1263.05 rows=1 width=1408) (actual time=4.832..4.835 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 33kB
                                      Buffers: shared hit=622
                                      ->  Result  (cost=634.20..1263.04 rows=1 width=1408) (actual time=4.220..4.809 rows=4 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=622
                                            ->  Hash Semi Join  (cost=634.20..1263.04 rows=1 width=1400) (actual time=4.205..4.791 rows=4 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=622
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.86..643.28 rows=155 width=1400) (actual time=0.207..1.286 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (actual time=0.111..0.112 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=3.373..3.373 rows=8098 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.021..1.833 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.387..0.388 rows=1 loops=1)
                                Buffers: shared hit=184
                                ->  Result  (cost=0.86..158.05 rows=1 width=1408) (actual time=0.385..0.386 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=184
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=1400) (actual time=0.352..0.353 rows=1 loops=1)
                                            Buffers: shared hit=184
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.026..0.105 rows=36 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=40
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=36)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=144
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.113..0.116 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1408) (actual time=0.112..0.115 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1408) (actual time=0.104..0.107 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1400) (actual time=0.100..0.102 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.061..0.079 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.046..0.048 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.038 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.023 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.006..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                        Filter: (value < '200'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.779 ms
Execution Time: 62.088 ms
```

### S2 num lt 2%, agents seeing 6% (control)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2640.91..2640.92 rows=1 width=8) (actual time=19.807..19.821 rows=1 loops=1)
  Buffers: shared hit=16752
  ->  Limit  (cost=619.88..2640.17 rows=59 width=4) (actual time=3.379..19.750 rows=487 loops=1)
        Buffers: shared hit=16752
        ->  Result  (cost=619.88..2640.17 rows=59 width=4) (actual time=3.377..19.679 rows=487 loops=1)
              Buffers: shared hit=16752
              ->  Append  (cost=619.88..2639.58 rows=59 width=0) (actual time=3.377..19.585 rows=487 loops=1)
                    Buffers: shared hit=16752
                    ->  Subquery Scan on "*SELECT* 1"  (cost=619.88..1172.82 rows=56 width=0) (actual time=3.375..14.309 rows=481 loops=1)
                          Buffers: shared hit=15782
                          ->  Limit  (cost=619.88..1172.26 rows=56 width=4) (actual time=3.374..14.230 rows=481 loops=1)
                                Buffers: shared hit=15782
                                ->  Result  (cost=619.88..1172.26 rows=56 width=4) (actual time=3.373..14.168 rows=481 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=15782
                                      ->  Hash Semi Join  (cost=619.88..1172.26 rows=56 width=0) (actual time=3.364..14.080 rows=481 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15782
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.028..6.792 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=3.323..3.324 rows=8098 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.042..1.826 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 2"  (cost=634.20..1263.05 rows=1 width=0) (actual time=3.993..4.577 rows=4 loops=1)
                          Buffers: shared hit=622
                          ->  Limit  (cost=634.20..1263.04 rows=1 width=4) (actual time=3.991..4.573 rows=4 loops=1)
                                Buffers: shared hit=622
                                ->  Result  (cost=634.20..1263.04 rows=1 width=4) (actual time=3.989..4.570 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=622
                                      ->  Hash Semi Join  (cost=634.20..1263.04 rows=1 width=0) (actual time=3.975..4.554 rows=4 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=622
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.86..643.28 rows=155 width=32) (actual time=0.223..1.218 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (actual time=0.106..0.106 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=3.176..3.177 rows=8098 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.020..1.713 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.06 rows=1 width=0) (actual time=0.333..0.523 rows=2 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..158.05 rows=1 width=4) (actual time=0.332..0.522 rows=2 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.05 rows=1 width=4) (actual time=0.330..0.519 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.05 rows=1 width=0) (actual time=0.320..0.508 rows=2 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (actual time=0.023..0.135 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (actual time=0.093..0.095 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (actual time=0.092..0.094 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (actual time=0.092..0.093 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (actual time=0.089..0.091 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.056..0.072 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.043..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.080 ms
Execution Time: 19.980 ms
```

### S2 num lt 2%, agents seeing 26%

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=53095.32..53253.89 rows=1 width=1450) (actual time=51.230..51.556 rows=51 loops=1)
  Buffers: shared hit=33642
  ->  Nested Loop  (cost=53095.02..53245.56 rows=1 width=1415) (actual time=51.215..51.398 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=33489
        ->  Subquery Scan on x  (cost=53094.74..53237.24 rows=1 width=1408) (actual time=51.205..51.285 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=33336
              ->  Limit  (cost=53094.74..53236.10 rows=51 width=1408) (actual time=51.202..51.248 rows=51 loops=1)
                    Buffers: shared hit=33336
                    ->  Merge Append  (cost=53094.74..53244.41 rows=54 width=1408) (actual time=51.201..51.241 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=33336
                          ->  Limit  (cost=51717.87..51717.99 rows=51 width=1408) (actual time=46.859..46.875 rows=51 loops=1)
                                Buffers: shared hit=32493
                                ->  Sort  (cost=51717.87..51718.48 rows=245 width=1408) (actual time=46.858..46.869 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 189kB
                                      Buffers: shared hit=32493
                                      ->  Result  (cost=564.32..51709.69 rows=245 width=1408) (actual time=3.955..45.301 rows=2107 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=32493
                                            ->  Nested Loop  (cost=564.32..51709.69 rows=245 width=1400) (actual time=3.949..44.663 rows=2107 loops=1)
                                                  Buffers: shared hit=32493
                                                  ->  HashAggregate  (cost=563.84..637.81 rows=7397 width=32) (actual time=3.914..5.993 rows=8098 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 1169kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.022..1.478 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                                                  ->  Index Scan using records_pkey on records r  (cost=0.46..6.93 rows=1 width=1400) (actual time=0.004..0.004 rows=0 loops=8098)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=32392
                          ->  Limit  (cost=1263.03..1263.04 rows=1 width=1408) (actual time=3.974..3.977 rows=1 loops=1)
                                Buffers: shared hit=622
                                ->  Sort  (cost=1263.03..1263.04 rows=1 width=1408) (actual time=3.972..3.974 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 33kB
                                      Buffers: shared hit=622
                                      ->  Result  (cost=634.23..1263.02 rows=1 width=1408) (actual time=3.470..3.956 rows=4 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=622
                                            ->  Hash Semi Join  (cost=634.23..1263.02 rows=1 width=1400) (actual time=3.461..3.944 rows=4 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=622
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.89..643.32 rows=139 width=1400) (actual time=0.174..1.022 rows=382 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                        Rows Removed by Filter: 131
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (actual time=0.091..0.092 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=2.841..2.842 rows=8098 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.016..1.521 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=0.90..149.62 rows=1 width=1408) (actual time=0.262..0.263 rows=1 loops=1)
                                Buffers: shared hit=168
                                ->  Result  (cost=0.90..149.62 rows=1 width=1408) (actual time=0.260..0.261 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=168
                                      ->  Nested Loop Semi Join  (cost=0.90..149.62 rows=1 width=1400) (actual time=0.254..0.254 rows=1 loops=1)
                                            Buffers: shared hit=168
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.46..56.55 rows=11 width=1400) (actual time=0.019..0.076 rows=32 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                  Rows Removed by Filter: 5
                                                  Buffers: shared hit=40
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=32)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=128
                          ->  Limit  (cost=112.90..112.91 rows=1 width=1408) (actual time=0.100..0.103 rows=0 loops=1)
                                Buffers: shared hit=53
                                ->  Sort  (cost=112.90..112.91 rows=1 width=1408) (actual time=0.100..0.102 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=53
                                      ->  Result  (cost=104.33..112.89 rows=1 width=1408) (actual time=0.094..0.096 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=53
                                            ->  Nested Loop Semi Join  (cost=104.33..112.89 rows=1 width=1400) (actual time=0.091..0.093 rows=0 loops=1)
                                                  Buffers: shared hit=53
                                                  ->  Nested Loop  (cost=103.90..111.94 rows=1 width=1416) (actual time=0.066..0.079 rows=2 loops=1)
                                                        Buffers: shared hit=45
                                                        ->  HashAggregate  (cost=103.44..103.45 rows=1 width=32) (actual time=0.054..0.056 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=33
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=60.46..103.43 rows=1 width=32) (actual time=0.045..0.049 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=33
                                                                    ->  BitmapOr  (cost=60.43..60.43 rows=13 width=0) (actual time=0.037..0.038 rows=0 loops=1)
                                                                          Buffers: shared hit=30
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..56.13 rows=12 width=0) (actual time=0.030..0.030 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                                                Buffers: shared hit=28
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.46..8.49 rows=1 width=1400) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                              Rows Removed by Filter: 0
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.93 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=2)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                        Filter: (value < '200'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.604 ms
Execution Time: 51.803 ms
```

### S2 num lt 2%, agents seeing 26%

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=4548.03..4548.04 rows=1 width=8) (actual time=22.453..22.461 rows=1 loops=1)
  Buffers: shared hit=31868
  ->  Limit  (cost=619.88..4544.93 rows=248 width=4) (actual time=2.973..22.342 rows=1001 loops=1)
        Buffers: shared hit=31868
        ->  Result  (cost=619.88..4544.93 rows=248 width=4) (actual time=2.972..22.215 rows=1001 loops=1)
              Buffers: shared hit=31868
              ->  Append  (cost=619.88..4542.45 rows=248 width=0) (actual time=2.972..22.050 rows=1001 loops=1)
                    Buffers: shared hit=31868
                    ->  Subquery Scan on "*SELECT* 1"  (cost=619.88..3015.64 rows=245 width=0) (actual time=2.971..21.914 rows=1001 loops=1)
                          Buffers: shared hit=31868
                          ->  Limit  (cost=619.88..3013.19 rows=245 width=4) (actual time=2.970..21.775 rows=1001 loops=1)
                                Buffers: shared hit=31868
                                ->  Result  (cost=619.88..3013.19 rows=245 width=4) (actual time=2.970..21.660 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=31868
                                      ->  Hash Semi Join  (cost=619.88..3013.19 rows=245 width=0) (actual time=2.963..21.508 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=31868
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..2321.62 rows=26481 width=32) (actual time=0.022..11.944 rows=48853 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=31767
                                            ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=2.931..2.932 rows=8098 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.015..1.571 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 2"  (cost=634.23..1263.03 rows=1 width=0) (never executed)
                          ->  Limit  (cost=634.23..1263.02 rows=1 width=4) (never executed)
                                ->  Result  (cost=634.23..1263.02 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Hash Semi Join  (cost=634.23..1263.02 rows=1 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.89..643.32 rows=139 width=32) (never executed)
                                                  Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (never executed)
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..526.86 rows=7397 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.90..149.63 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.90..149.62 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.90..149.62 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.90..149.62 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.46..56.55 rows=11 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=104.33..112.90 rows=1 width=0) (never executed)
                          ->  Limit  (cost=104.33..112.89 rows=1 width=4) (never executed)
                                ->  Result  (cost=104.33..112.89 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=104.33..112.89 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=103.90..111.94 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=103.44..103.45 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=60.46..103.43 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                              ->  BitmapOr  (cost=60.43..60.43 rows=13 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..56.13 rows=12 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.46..8.49 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.93 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.819 ms
Execution Time: 22.586 ms
```

### S2 num lt 2%, agents seeing 50%

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=53314.38..53464.96 rows=1 width=1450) (actual time=53.137..53.472 rows=51 loops=1)
  Buffers: shared hit=33689
  ->  Nested Loop  (cost=53314.08..53456.62 rows=1 width=1415) (actual time=53.123..53.307 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=33536
        ->  Subquery Scan on x  (cost=53313.80..53448.31 rows=1 width=1408) (actual time=53.112..53.195 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=33383
              ->  Limit  (cost=53313.80..53447.16 rows=51 width=1408) (actual time=53.110..53.158 rows=51 loops=1)
                    Buffers: shared hit=33383
                    ->  Merge Append  (cost=53313.80..53455.01 rows=54 width=1408) (actual time=53.108..53.150 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=33383
                          ->  Limit  (cost=51868.23..51868.36 rows=51 width=1408) (actual time=48.678..48.695 rows=51 loops=1)
                                Buffers: shared hit=32493
                                ->  Sort  (cost=51868.23..51869.39 rows=462 width=1408) (actual time=48.660..48.672 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 197kB
                                      Buffers: shared hit=32493
                                      ->  Result  (cost=564.35..51852.82 rows=462 width=1408) (actual time=3.742..45.786 rows=4062 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=32493
                                            ->  Nested Loop  (cost=564.35..51852.82 rows=462 width=1400) (actual time=3.738..44.598 rows=4062 loops=1)
                                                  Buffers: shared hit=32493
                                                  ->  HashAggregate  (cost=563.84..637.81 rows=7397 width=32) (actual time=3.699..5.641 rows=8098 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 1169kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.023..1.370 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                                                  ->  Index Scan using records_pkey on records r  (cost=0.49..6.96 rows=1 width=1400) (actual time=0.004..0.004 rows=1 loops=8098)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=32392
                          ->  Limit  (cost=1263.01..1263.01 rows=1 width=1408) (actual time=3.998..4.001 rows=1 loops=1)
                                Buffers: shared hit=622
                                ->  Sort  (cost=1263.01..1263.01 rows=1 width=1408) (actual time=3.996..3.998 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=622
                                      ->  Result  (cost=634.25..1263.00 rows=1 width=1408) (actual time=3.490..3.983 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=622
                                            ->  Hash Semi Join  (cost=634.25..1263.00 rows=1 width=1400) (actual time=3.481..3.971 rows=3 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=622
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.92..643.34 rows=120 width=1400) (actual time=0.176..1.031 rows=256 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                        Rows Removed by Filter: 257
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (actual time=0.088..0.089 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=2.876..2.877 rows=8098 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.018..1.546 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=0.93..141.19 rows=1 width=1408) (actual time=0.305..0.306 rows=0 loops=1)
                                Buffers: shared hit=195
                                ->  Result  (cost=0.93..141.19 rows=1 width=1408) (actual time=0.303..0.305 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=195
                                      ->  Nested Loop Semi Join  (cost=0.93..141.19 rows=1 width=1400) (actual time=0.296..0.297 rows=0 loops=1)
                                            Buffers: shared hit=195
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.49..56.58 rows=10 width=1400) (actual time=0.020..0.115 rows=32 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                  Rows Removed by Filter: 32
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=32)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=128
                          ->  Limit  (cost=181.59..181.59 rows=1 width=1408) (actual time=0.122..0.124 rows=0 loops=1)
                                Buffers: shared hit=73
                                ->  Sort  (cost=181.59..181.59 rows=1 width=1408) (actual time=0.121..0.123 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=73
                                      ->  Result  (cost=172.95..181.58 rows=1 width=1408) (actual time=0.116..0.118 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=73
                                            ->  Nested Loop Semi Join  (cost=172.95..181.58 rows=1 width=1400) (actual time=0.113..0.115 rows=0 loops=1)
                                                  Buffers: shared hit=73
                                                  ->  Nested Loop  (cost=172.51..180.55 rows=1 width=1416) (actual time=0.099..0.106 rows=1 loops=1)
                                                        Buffers: shared hit=69
                                                        ->  HashAggregate  (cost=172.02..172.03 rows=1 width=32) (actual time=0.081..0.082 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=57
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=104.03..172.01 rows=1 width=32) (actual time=0.070..0.074 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=57
                                                                    ->  BitmapOr  (cost=103.97..103.97 rows=23 width=0) (actual time=0.063..0.064 rows=0 loops=1)
                                                                          Buffers: shared hit=54
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..99.67 rows=22 width=0) (actual time=0.056..0.056 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                                                Buffers: shared hit=52
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.49..8.52 rows=1 width=1400) (actual time=0.007..0.007 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.00 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                        Filter: (value < '200'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=4
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.551 ms
Execution Time: 53.717 ms
```

### S2 num lt 2%, agents seeing 50%

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=6650.57..6650.58 rows=1 width=8) (actual time=21.903..21.914 rows=1 loops=1)
  Buffers: shared hit=31509
  ->  Limit  (cost=619.88..6644.76 rows=465 width=4) (actual time=2.780..21.796 rows=1001 loops=1)
        Buffers: shared hit=31509
        ->  Result  (cost=619.88..6644.76 rows=465 width=4) (actual time=2.779..21.658 rows=1001 loops=1)
              Buffers: shared hit=31509
              ->  Append  (cost=619.88..6640.11 rows=465 width=0) (actual time=2.778..21.495 rows=1001 loops=1)
                    Buffers: shared hit=31509
                    ->  Subquery Scan on "*SELECT* 1"  (cost=619.88..5051.98 rows=462 width=0) (actual time=2.777..21.359 rows=1001 loops=1)
                          Buffers: shared hit=31509
                          ->  Limit  (cost=619.88..5047.36 rows=462 width=4) (actual time=2.776..21.211 rows=1001 loops=1)
                                Buffers: shared hit=31509
                                ->  Result  (cost=619.88..5047.36 rows=462 width=4) (actual time=2.775..21.090 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=31509
                                      ->  Hash Semi Join  (cost=619.88..5047.36 rows=462 width=0) (actual time=2.769..20.939 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=31509
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..4291.69 rows=49981 width=32) (actual time=0.023..11.589 rows=48383 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=31408
                                            ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (actual time=2.736..2.737 rows=8098 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..526.86 rows=7397 width=32) (actual time=0.013..1.463 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 2"  (cost=634.25..1263.01 rows=1 width=0) (never executed)
                          ->  Limit  (cost=634.25..1263.00 rows=1 width=4) (never executed)
                                ->  Result  (cost=634.25..1263.00 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Hash Semi Join  (cost=634.25..1263.00 rows=1 width=0) (never executed)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.92..643.34 rows=120 width=32) (never executed)
                                                  Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            ->  Hash  (cost=526.86..526.86 rows=7397 width=32) (never executed)
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..526.86 rows=7397 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.93..141.20 rows=1 width=0) (never executed)
                          ->  Limit  (cost=0.93..141.19 rows=1 width=4) (never executed)
                                ->  Result  (cost=0.93..141.19 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.93..141.19 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.49..56.58 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_2.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=172.95..181.59 rows=1 width=0) (never executed)
                          ->  Limit  (cost=172.95..181.58 rows=1 width=4) (never executed)
                                ->  Result  (cost=172.95..181.58 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=172.95..181.58 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=172.51..180.55 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=172.02..172.03 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=104.03..172.01 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                              ->  BitmapOr  (cost=103.97..103.97 rows=23 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..99.67 rows=22 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.49..8.52 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.00 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (record_id = r_3.id) AND (field_id = '1595509d-6ead-44cc-91b2-6717c8066fa5'::uuid))
                                                  Filter: (value < '200'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.766 ms
Execution Time: 22.043 ms
```

### S2 text eq 2%, agents seeing 6% (control)

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22831.23..22953.78 rows=1 width=1450) (actual time=51.857..52.199 rows=51 loops=1)
  Buffers: shared hit=23178
  ->  Nested Loop  (cost=22830.93..22945.45 rows=1 width=1415) (actual time=51.842..52.049 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=23025
        ->  Subquery Scan on x  (cost=22830.65..22937.13 rows=1 width=1408) (actual time=51.832..51.932 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22872
              ->  Limit  (cost=22830.65..22935.99 rows=51 width=1408) (actual time=51.830..51.877 rows=51 loops=1)
                    Buffers: shared hit=22872
                    ->  Merge Append  (cost=22830.65..22942.18 rows=54 width=1408) (actual time=51.828..51.870 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22872
                          ->  Limit  (cost=21485.17..21485.30 rows=51 width=1408) (actual time=47.150..47.169 rows=51 loops=1)
                                Buffers: shared hit=21900
                                ->  Sort  (cost=21485.17..21485.32 rows=57 width=1408) (actual time=47.149..47.162 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=21900
                                      ->  Result  (cost=1133.10..21483.51 rows=57 width=1408) (actual time=12.420..46.731 rows=493 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=21900
                                            ->  Hash Semi Join  (cost=1133.10..21483.51 rows=57 width=1400) (actual time=12.415..46.541 rows=493 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21900
                                                  ->  Bitmap Heap Scan on records r  (cost=477.88..20811.80 rows=6014 width=1400) (actual time=9.059..38.861 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..476.38 rows=6014 width=0) (actual time=4.106..4.106 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=558.55..558.55 rows=7733 width=32) (actual time=3.251..3.251 rows=7980 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                        Buffers: shared hit=103
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.026..1.761 rows=7980 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=103
                          ->  Limit  (cost=1298.93..1298.94 rows=1 width=1408) (actual time=4.209..4.211 rows=1 loops=1)
                                Buffers: shared hit=624
                                ->  Sort  (cost=1298.93..1298.94 rows=1 width=1408) (actual time=4.207..4.209 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 37kB
                                      Buffers: shared hit=624
                                      ->  Result  (cost=670.08..1298.92 rows=1 width=1408) (actual time=3.497..4.191 rows=6 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=624
                                            ->  Hash Semi Join  (cost=670.08..1298.92 rows=1 width=1400) (actual time=3.485..4.176 rows=6 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=624
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.86..643.28 rows=155 width=1400) (actual time=0.175..1.048 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (actual time=0.094..0.094 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=558.55..558.55 rows=7733 width=32) (actual time=3.024..3.024 rows=7980 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                        Buffers: shared hit=103
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_1  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.021..1.651 rows=7980 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=103
                          ->  Limit  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.374..0.375 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.58 rows=1 width=1408) (actual time=0.372..0.373 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=1400) (actual time=0.365..0.365 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=1400) (actual time=0.019..0.122 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.089..0.091 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.51..45.51 rows=1 width=1408) (actual time=0.088..0.091 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1408) (actual time=0.083..0.085 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1400) (actual time=0.080..0.082 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1416) (actual time=0.050..0.066 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.036..0.038 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.025..0.030 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.018..0.019 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.011 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1400) (actual time=0.008..0.008 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.004..0.005 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.965 ms
Execution Time: 52.455 ms
```

### S2 text eq 2%, agents seeing 6% (control)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2592.25..2592.26 rows=1 width=8) (actual time=16.842..16.852 rows=1 loops=1)
  Buffers: shared hit=16756
  ->  Limit  (cost=655.77..2591.50 rows=60 width=4) (actual time=3.095..16.787 rows=499 loops=1)
        Buffers: shared hit=16756
        ->  Result  (cost=655.77..2591.50 rows=60 width=4) (actual time=3.094..16.721 rows=499 loops=1)
              Buffers: shared hit=16756
              ->  Append  (cost=655.77..2590.90 rows=60 width=0) (actual time=3.093..16.632 rows=499 loops=1)
                    Buffers: shared hit=16756
                    ->  Subquery Scan on "*SELECT* 1"  (cost=655.77..1208.80 rows=57 width=0) (actual time=3.092..12.852 rows=493 loops=1)
                          Buffers: shared hit=15784
                          ->  Limit  (cost=655.77..1208.23 rows=57 width=4) (actual time=3.091..12.779 rows=493 loops=1)
                                Buffers: shared hit=15784
                                ->  Result  (cost=655.77..1208.23 rows=57 width=4) (actual time=3.090..12.720 rows=493 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=15784
                                      ->  Hash Semi Join  (cost=655.77..1208.23 rows=57 width=0) (actual time=3.081..12.639 rows=493 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15784
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..536.52 rows=6014 width=32) (actual time=0.025..6.122 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=558.55..558.55 rows=7733 width=32) (actual time=3.020..3.020 rows=7980 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                  Buffers: shared hit=103
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.020..1.650 rows=7980 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=103
                    ->  Subquery Scan on "*SELECT* 2"  (cost=649.49..1224.70 rows=1 width=0) (actual time=2.153..3.283 rows=6 loops=1)
                          Buffers: shared hit=624
                          ->  Limit  (cost=649.49..1224.69 rows=1 width=4) (actual time=2.152..3.280 rows=6 loops=1)
                                Buffers: shared hit=624
                                ->  Result  (cost=649.49..1224.69 rows=1 width=4) (actual time=2.150..3.277 rows=6 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=624
                                      ->  Merge Semi Join  (cost=649.49..1224.69 rows=1 width=0) (actual time=2.141..3.266 rows=6 loops=1)
                                            Merge Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=624
                                            ->  Sort  (cost=648.92..649.31 rows=155 width=32) (actual time=1.172..1.206 rows=484 loops=1)
                                                  Sort Key: r_1.id
                                                  Sort Method: quicksort  Memory: 51kB
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.86..643.28 rows=155 width=32) (actual time=0.198..1.035 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (actual time=0.093..0.093 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.019..1.323 rows=7980 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=103
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.59 rows=1 width=0) (actual time=0.341..0.342 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..111.58 rows=1 width=4) (actual time=0.340..0.341 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.58 rows=1 width=4) (actual time=0.339..0.340 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.58 rows=1 width=0) (actual time=0.332..0.333 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.52 rows=12 width=32) (actual time=0.017..0.121 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (actual time=0.082..0.085 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (actual time=0.081..0.084 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (actual time=0.081..0.083 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (actual time=0.078..0.081 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.049..0.065 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.040 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.026..0.032 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.007 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.936 ms
Execution Time: 16.994 ms
```

### S2 text eq 2%, agents seeing 26%

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=53735.31..55071.43 rows=1 width=1450) (actual time=47.731..48.054 rows=51 loops=1)
  Buffers: shared hit=33029
  ->  Nested Loop  (cost=53735.01..55063.10 rows=1 width=1415) (actual time=47.716..47.897 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=32876
        ->  Subquery Scan on x  (cost=53734.73..55054.79 rows=1 width=1408) (actual time=47.705..47.784 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=32723
              ->  Limit  (cost=53734.73..55053.64 rows=51 width=1408) (actual time=47.702..47.747 rows=51 loops=1)
                    Buffers: shared hit=32723
                    ->  Merge Append  (cost=53734.73..55131.22 rows=54 width=1408) (actual time=47.701..47.740 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=32723
                          ->  Limit  (cost=53619.59..53619.72 rows=51 width=1408) (actual time=46.795..46.813 rows=51 loops=1)
                                Buffers: shared hit=32023
                                ->  Sort  (cost=53619.59..53620.22 rows=252 width=1408) (actual time=46.794..46.806 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 199kB
                                      Buffers: shared hit=32023
                                      ->  Result  (cost=597.68..53611.18 rows=252 width=1408) (actual time=3.826..45.218 rows=2124 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=32023
                                            ->  Nested Loop  (cost=597.68..53611.18 rows=252 width=1400) (actual time=3.821..44.559 rows=2124 loops=1)
                                                  Buffers: shared hit=32023
                                                  ->  HashAggregate  (cost=597.21..674.36 rows=7715 width=32) (actual time=3.795..5.886 rows=7980 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 1169kB
                                                        Buffers: shared hit=103
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.031..1.429 rows=7980 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=103
                                                  ->  Index Scan using records_pkey on records r  (cost=0.46..6.88 rows=1 width=1400) (actual time=0.005..0.005 rows=0 loops=7980)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=31920
                          ->  Limit  (cost=1.02..1290.58 rows=1 width=1408) (actual time=0.454..0.455 rows=1 loops=1)
                                Buffers: shared hit=372
                                ->  Result  (cost=1.02..1290.58 rows=1 width=1408) (actual time=0.452..0.453 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=372
                                      ->  Nested Loop Semi Join  (cost=1.02..1290.58 rows=1 width=1400) (actual time=0.442..0.442 rows=1 loops=1)
                                            Buffers: shared hit=372
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.46..652.90 rows=139 width=1400) (actual time=0.019..0.170 rows=68 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                  Rows Removed by Filter: 27
                                                  Buffers: shared hit=99
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=68)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=273
                          ->  Limit  (cost=1.02..107.03 rows=1 width=1408) (actual time=0.336..0.337 rows=0 loops=1)
                                Buffers: shared hit=275
                                ->  Result  (cost=1.02..107.03 rows=1 width=1408) (actual time=0.335..0.336 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=275
                                      ->  Nested Loop Semi Join  (cost=1.02..107.03 rows=1 width=1400) (actual time=0.332..0.333 rows=0 loops=1)
                                            Buffers: shared hit=275
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.46..56.55 rows=11 width=1400) (actual time=0.015..0.121 rows=52 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                  Rows Removed by Filter: 12
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=52)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=208
                          ->  Limit  (cost=113.05..113.05 rows=1 width=1408) (actual time=0.110..0.113 rows=0 loops=1)
                                Buffers: shared hit=53
                                ->  Sort  (cost=113.05..113.05 rows=1 width=1408) (actual time=0.109..0.112 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=53
                                      ->  Result  (cost=104.46..113.04 rows=1 width=1408) (actual time=0.102..0.104 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=53
                                            ->  Nested Loop Semi Join  (cost=104.46..113.04 rows=1 width=1400) (actual time=0.099..0.101 rows=0 loops=1)
                                                  Buffers: shared hit=53
                                                  ->  Nested Loop  (cost=103.90..111.94 rows=1 width=1416) (actual time=0.076..0.089 rows=2 loops=1)
                                                        Buffers: shared hit=45
                                                        ->  HashAggregate  (cost=103.44..103.45 rows=1 width=32) (actual time=0.063..0.065 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=33
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=60.46..103.43 rows=1 width=32) (actual time=0.053..0.058 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=33
                                                                    ->  BitmapOr  (cost=60.43..60.43 rows=13 width=0) (actual time=0.044..0.045 rows=0 loops=1)
                                                                          Buffers: shared hit=30
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..56.13 rows=12 width=0) (actual time=0.034..0.034 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                                                Buffers: shared hit=28
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.46..8.49 rows=1 width=1400) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                              Rows Removed by Filter: 0
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.08 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.882 ms
Execution Time: 48.334 ms
```

### S2 text eq 2%, agents seeing 26%

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=4503.05..4503.06 rows=1 width=8) (actual time=22.894..22.906 rows=1 loops=1)
  Buffers: shared hit=31636
  ->  Limit  (cost=655.77..4499.86 rows=255 width=4) (actual time=3.046..22.786 rows=1001 loops=1)
        Buffers: shared hit=31636
        ->  Result  (cost=655.77..4499.86 rows=255 width=4) (actual time=3.045..22.660 rows=1001 loops=1)
              Buffers: shared hit=31636
              ->  Append  (cost=655.77..4497.31 rows=255 width=0) (actual time=3.044..22.490 rows=1001 loops=1)
                    Buffers: shared hit=31636
                    ->  Subquery Scan on "*SELECT* 1"  (cost=655.77..3051.99 rows=252 width=0) (actual time=3.043..22.347 rows=1001 loops=1)
                          Buffers: shared hit=31636
                          ->  Limit  (cost=655.77..3049.47 rows=252 width=4) (actual time=3.042..22.199 rows=1001 loops=1)
                                Buffers: shared hit=31636
                                ->  Result  (cost=655.77..3049.47 rows=252 width=4) (actual time=3.041..22.073 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=31636
                                      ->  Hash Semi Join  (cost=655.77..3049.47 rows=252 width=0) (actual time=3.035..21.920 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=31636
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..2321.62 rows=26481 width=32) (actual time=0.023..12.186 rows=48513 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=31533
                                            ->  Hash  (cost=558.55..558.55 rows=7733 width=32) (actual time=2.980..2.981 rows=7980 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                  Buffers: shared hit=103
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.016..1.635 rows=7980 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=103
                    ->  Subquery Scan on "*SELECT* 2"  (cost=648.83..1223.96 rows=1 width=0) (never executed)
                          ->  Limit  (cost=648.83..1223.95 rows=1 width=4) (never executed)
                                ->  Result  (cost=648.83..1223.95 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Merge Semi Join  (cost=648.83..1223.95 rows=1 width=0) (never executed)
                                            Merge Cond: (r_1.id = v_1.record_id)
                                            ->  Sort  (cost=648.26..648.61 rows=139 width=32) (never executed)
                                                  Sort Key: r_1.id
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.89..643.32 rows=139 width=32) (never executed)
                                                        Recheck Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.83 rows=160 width=0) (never executed)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..558.55 rows=7733 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.02..107.04 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.02..107.03 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.02..107.03 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.02..107.03 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.46..56.55 rows=11 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=104.46..113.05 rows=1 width=0) (never executed)
                          ->  Limit  (cost=104.46..113.04 rows=1 width=4) (never executed)
                                ->  Result  (cost=104.46..113.04 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=104.46..113.04 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=103.90..111.94 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=103.44..103.45 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=60.46..103.43 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                                              ->  BitmapOr  (cost=60.43..60.43 rows=13 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..56.13 rows=12 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.46..8.49 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,3832b035-38e3-c5dd-ff19-06633840e6e6,42a4e205-033a-0571-9528-b4862c070617,3380ce13-07ca-f993-be7f-051e91a24557,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,7e4b428f-5e9c-b350-5822-954af8729d3a,bc877580-07cd-8d87-e29c-e7008bd74f9e,b4867584-2d37-1b65-7e62-c7eb8682704e,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,9cedca87-6355-9ae5-590f-e07dcb7ecb8f}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.08 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 1.988 ms
Execution Time: 23.034 ms
```

### S2 text eq 2%, agents seeing 50%

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.org_unit_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.org_unit_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=53956.52..55205.99 rows=1 width=1450) (actual time=54.171..54.504 rows=51 loops=1)
  Buffers: shared hit=33102
  ->  Nested Loop  (cost=53956.22..55197.66 rows=1 width=1415) (actual time=54.156..54.339 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=32949
        ->  Subquery Scan on x  (cost=53955.94..55189.34 rows=1 width=1408) (actual time=54.147..54.227 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=32796
              ->  Limit  (cost=53955.94..55188.20 rows=51 width=1408) (actual time=54.144..54.189 rows=51 loops=1)
                    Buffers: shared hit=32796
                    ->  Merge Append  (cost=53955.94..55260.68 rows=54 width=1408) (actual time=54.143..54.182 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=32796
                          ->  Limit  (cost=53772.05..53772.18 rows=51 width=1408) (actual time=53.128..53.147 rows=51 loops=1)
                                Buffers: shared hit=32023
                                ->  Sort  (cost=53772.05..53773.24 rows=476 width=1408) (actual time=53.127..53.140 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 205kB
                                      Buffers: shared hit=32023
                                      ->  Result  (cost=597.71..53756.17 rows=476 width=1408) (actual time=4.035..50.310 rows=4002 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=32023
                                            ->  Nested Loop  (cost=597.71..53756.17 rows=476 width=1400) (actual time=4.030..49.176 rows=4002 loops=1)
                                                  Buffers: shared hit=32023
                                                  ->  HashAggregate  (cost=597.21..674.36 rows=7715 width=32) (actual time=4.006..5.895 rows=7980 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 1169kB
                                                        Buffers: shared hit=103
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.029..1.556 rows=7980 loops=1)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=103
                                                  ->  Index Scan using records_pkey on records r  (cost=0.49..6.91 rows=1 width=1400) (actual time=0.005..0.005 rows=1 loops=7980)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=31920
                          ->  Limit  (cost=1.05..1203.44 rows=1 width=1408) (actual time=0.626..0.627 rows=1 loops=1)
                                Buffers: shared hit=505
                                ->  Result  (cost=1.05..1203.44 rows=1 width=1408) (actual time=0.624..0.625 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=505
                                      ->  Nested Loop Semi Join  (cost=1.05..1203.44 rows=1 width=1400) (actual time=0.611..0.612 rows=1 loops=1)
                                            Buffers: shared hit=505
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.49..652.93 rows=120 width=1400) (actual time=0.029..0.294 rows=80 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                  Rows Removed by Filter: 98
                                                  Buffers: shared hit=184
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=80)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=321
                          ->  Limit  (cost=1.05..102.47 rows=1 width=1408) (actual time=0.231..0.232 rows=0 loops=1)
                                Buffers: shared hit=195
                                ->  Result  (cost=1.05..102.47 rows=1 width=1408) (actual time=0.230..0.231 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=195
                                      ->  Nested Loop Semi Join  (cost=1.05..102.47 rows=1 width=1400) (actual time=0.227..0.228 rows=0 loops=1)
                                            Buffers: shared hit=195
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.49..56.58 rows=10 width=1400) (actual time=0.015..0.109 rows=32 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                  Rows Removed by Filter: 32
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=32)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=128
                          ->  Limit  (cost=181.74..181.74 rows=1 width=1408) (actual time=0.152..0.154 rows=0 loops=1)
                                Buffers: shared hit=73
                                ->  Sort  (cost=181.74..181.74 rows=1 width=1408) (actual time=0.151..0.153 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=73
                                      ->  Result  (cost=173.07..181.73 rows=1 width=1408) (actual time=0.124..0.126 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                            Buffers: shared hit=73
                                            ->  Nested Loop Semi Join  (cost=173.07..181.73 rows=1 width=1400) (actual time=0.120..0.122 rows=0 loops=1)
                                                  Buffers: shared hit=73
                                                  ->  Nested Loop  (cost=172.51..180.55 rows=1 width=1416) (actual time=0.106..0.113 rows=1 loops=1)
                                                        Buffers: shared hit=69
                                                        ->  HashAggregate  (cost=172.02..172.03 rows=1 width=32) (actual time=0.086..0.088 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=57
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=104.03..172.01 rows=1 width=32) (actual time=0.075..0.080 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=57
                                                                    ->  BitmapOr  (cost=103.97..103.97 rows=23 width=0) (actual time=0.067..0.067 rows=0 loops=1)
                                                                          Buffers: shared hit=54
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..99.67 rows=22 width=0) (actual time=0.056..0.056 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                                                Buffers: shared hit=52
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.49..8.52 rows=1 width=1400) (actual time=0.007..0.007 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.16 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 3.076 ms
Execution Time: 54.743 ms
```

### S2 text eq 2%, agents seeing 50%

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=6589.60..6589.61 rows=1 width=8) (actual time=22.479..22.487 rows=1 loops=1)
  Buffers: shared hit=32499
  ->  Limit  (cost=655.77..6583.62 rows=479 width=4) (actual time=3.059..22.369 rows=1001 loops=1)
        Buffers: shared hit=32499
        ->  Result  (cost=655.77..6583.62 rows=479 width=4) (actual time=3.058..22.244 rows=1001 loops=1)
              Buffers: shared hit=32499
              ->  Append  (cost=655.77..6578.83 rows=479 width=0) (actual time=3.058..22.082 rows=1001 loops=1)
                    Buffers: shared hit=32499
                    ->  Subquery Scan on "*SELECT* 1"  (cost=655.77..5088.76 rows=476 width=0) (actual time=3.057..21.948 rows=1001 loops=1)
                          Buffers: shared hit=32499
                          ->  Limit  (cost=655.77..5084.00 rows=476 width=4) (actual time=3.056..21.807 rows=1001 loops=1)
                                Buffers: shared hit=32499
                                ->  Result  (cost=655.77..5084.00 rows=476 width=4) (actual time=3.055..21.693 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      Buffers: shared hit=32499
                                      ->  Hash Semi Join  (cost=655.77..5084.00 rows=476 width=0) (actual time=3.050..21.548 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=32499
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..4291.69 rows=49981 width=32) (actual time=0.027..12.003 rows=49906 loops=1)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=32396
                                            ->  Hash  (cost=558.55..558.55 rows=7733 width=32) (actual time=3.006..3.007 rows=7980 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                  Buffers: shared hit=103
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..558.55 rows=7733 width=32) (actual time=0.018..1.673 rows=7980 loops=1)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=103
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.05..1203.45 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.05..1203.44 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.05..1203.44 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.05..1203.44 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.49..652.93 rows=120 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.05..102.48 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.05..102.47 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.05..102.47 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.05..102.47 rows=1 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.49..56.58 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=173.07..181.74 rows=1 width=0) (never executed)
                          ->  Limit  (cost=173.07..181.73 rows=1 width=4) (never executed)
                                ->  Result  (cost=173.07..181.73 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid)
                                      ->  Nested Loop Semi Join  (cost=173.07..181.73 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=172.51..180.55 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=172.02..172.03 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=104.03..172.01 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                                              ->  BitmapOr  (cost=103.97..103.97 rows=23 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..99.67 rows=22 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.49..8.52 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a21f7a-3cc7-7bf7-b010-4f7d0e8042f7'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,af582227-36de-d5f3-b486-23b9d5f5d1a8,3832b035-38e3-c5dd-ff19-06633840e6e6,7a12a0cb-7a56-3ceb-ba65-6f71924aadb8,42a4e205-033a-0571-9528-b4862c070617,94b929ae-0527-95f2-8d7d-ea116ea94e4e,3380ce13-07ca-f993-be7f-051e91a24557,ebb73e90-c65e-cc46-8fce-e9035015a55d,0177f8b5-83ff-1f47-1ebb-c8eb08e44ef7,2f067c2d-b39a-c447-6bd8-6f26d53e8105,76cc839e-388e-0aab-c8f9-290e29a5b2c9,7e4b428f-5e9c-b350-5822-954af8729d3a,ee698b54-7dbd-325e-f543-70d400db98c5,bc877580-07cd-8d87-e29c-e7008bd74f9e,062e94b7-d3f5-e915-8445-d0b556a00eba,b4867584-2d37-1b65-7e62-c7eb8682704e,2e43eb92-0a43-7054-b645-b6740d31bf6d,d6fbf387-c222-9213-ed7c-a51046a0fa35,e43d6642-9bbe-05be-b6a6-8d3a87bf4141,60544299-50cd-24ce-634e-edc6cc80d735,47a078ad-c1c7-76f3-7a43-a34e987a2f43,9cedca87-6355-9ae5-590f-e07dcb7ecb8f,a0461d4f-0898-88fd-8c23-9e3da6d272e9,2f28f53e-68bf-1249-b1b6-271878028981,fcb88798-7208-926a-2cd2-13ba3946b83b}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.16 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '8ae29651-4fa1-4493-8a3d-9aae9371c7fa'::uuid) AND (field_id = '30536b46-20e9-4a86-93d4-bde9618e294a'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning:
  Buffers: shared hit=4
Planning Time: 2.034 ms
Execution Time: 22.616 ms
```
