# TICK-1048 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)

### attempt 1: band (ii) kp=3, agents, count skipped

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=48.24..9637.84 rows=1 width=1317) (actual time=0.523..3.774 rows=51 loops=1)
  Buffers: shared hit=2253
  ->  Nested Loop  (cost=47.94..9629.50 rows=1 width=1282) (actual time=0.503..3.568 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2100
        ->  Subquery Scan on x  (cost=47.66..9621.19 rows=1 width=1275) (actual time=0.489..3.414 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1947
              ->  Limit  (cost=47.66..9620.04 rows=51 width=1275) (actual time=0.486..3.334 rows=51 loops=1)
                    Buffers: shared hit=1947
                    ->  Merge Append  (cost=47.66..15438.55 rows=82 width=1275) (actual time=0.485..3.324 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1947
                          ->  Limit  (cost=0.86..14405.28 rows=51 width=1275) (actual time=0.261..3.076 rows=51 loops=1)
                                Buffers: shared hit=1892
                                ->  Result  (cost=0.86..410103.04 rows=1452 width=1275) (actual time=0.261..3.066 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=1892
                                      ->  Nested Loop Semi Join  (cost=0.86..410103.04 rows=1452 width=1267) (actual time=0.256..3.042 rows=51 loops=1)
                                            Buffers: shared hit=1892
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..392312.87 rows=6055 width=1267) (actual time=0.131..2.466 rows=100 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1470
                                                  Buffers: shared hit=1591
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.93 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=100)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=301
                          ->  Limit  (cost=0.86..881.19 rows=27 width=1275) (actual time=0.049..0.051 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=0.86..881.19 rows=27 width=1275) (actual time=0.048..0.050 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=12
                                      ->  Nested Loop Semi Join  (cost=0.86..881.19 rows=27 width=1267) (actual time=0.044..0.045 rows=1 loops=1)
                                            Buffers: shared hit=12
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..385.94 rows=112 width=1267) (actual time=0.019..0.022 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.41 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=7
                          ->  Limit  (cost=0.86..105.76 rows=3 width=1275) (actual time=0.035..0.037 rows=1 loops=1)
                                Buffers: shared hit=8
                                ->  Result  (cost=0.86..105.76 rows=3 width=1275) (actual time=0.035..0.036 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=8
                                      ->  Nested Loop Semi Join  (cost=0.86..105.76 rows=3 width=1267) (actual time=0.033..0.034 rows=1 loops=1)
                                            Buffers: shared hit=8
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..47.80 rows=13 width=1267) (actual time=0.021..0.021 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=4
                          ->  Limit  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.137..0.142 rows=1 loops=1)
                                Buffers: shared hit=35
                                ->  Sort  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.137..0.142 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=35
                                      ->  Result  (cost=36.84..45.03 rows=1 width=1275) (actual time=0.088..0.128 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                            Buffers: shared hit=35
                                            ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=1267) (actual time=0.086..0.125 rows=3 loops=1)
                                                  Buffers: shared hit=35
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1283) (actual time=0.070..0.093 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.051..0.054 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.045 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.032..0.032 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.011..0.011 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=10
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.779 ms
Execution Time: 3.991 ms
```

### attempt 1: band (ii) kp=3, agents, with the budgeted count

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=48.24..9637.84 rows=1 width=1317) (actual time=0.413..3.672 rows=51 loops=1)
  Buffers: shared hit=2253
  ->  Nested Loop  (cost=47.94..9629.50 rows=1 width=1282) (actual time=0.399..3.478 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2100
        ->  Subquery Scan on x  (cost=47.66..9621.19 rows=1 width=1275) (actual time=0.390..3.335 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1947
              ->  Limit  (cost=47.66..9620.04 rows=51 width=1275) (actual time=0.388..3.280 rows=51 loops=1)
                    Buffers: shared hit=1947
                    ->  Merge Append  (cost=47.66..15438.55 rows=82 width=1275) (actual time=0.387..3.269 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1947
                          ->  Limit  (cost=0.86..14405.28 rows=51 width=1275) (actual time=0.211..3.068 rows=51 loops=1)
                                Buffers: shared hit=1892
                                ->  Result  (cost=0.86..410103.04 rows=1452 width=1275) (actual time=0.211..3.058 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=1892
                                      ->  Nested Loop Semi Join  (cost=0.86..410103.04 rows=1452 width=1267) (actual time=0.206..3.034 rows=51 loops=1)
                                            Buffers: shared hit=1892
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..392312.87 rows=6055 width=1267) (actual time=0.093..2.476 rows=100 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1470
                                                  Buffers: shared hit=1591
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.93 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=100)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=301
                          ->  Limit  (cost=0.86..881.19 rows=27 width=1275) (actual time=0.039..0.041 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=0.86..881.19 rows=27 width=1275) (actual time=0.039..0.040 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=12
                                      ->  Nested Loop Semi Join  (cost=0.86..881.19 rows=27 width=1267) (actual time=0.036..0.037 rows=1 loops=1)
                                            Buffers: shared hit=12
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..385.94 rows=112 width=1267) (actual time=0.015..0.019 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.41 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=7
                          ->  Limit  (cost=0.86..105.76 rows=3 width=1275) (actual time=0.024..0.026 rows=1 loops=1)
                                Buffers: shared hit=8
                                ->  Result  (cost=0.86..105.76 rows=3 width=1275) (actual time=0.024..0.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=8
                                      ->  Nested Loop Semi Join  (cost=0.86..105.76 rows=3 width=1267) (actual time=0.021..0.023 rows=1 loops=1)
                                            Buffers: shared hit=8
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..47.80 rows=13 width=1267) (actual time=0.015..0.015 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=4
                          ->  Limit  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.110..0.116 rows=1 loops=1)
                                Buffers: shared hit=35
                                ->  Sort  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.110..0.115 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=35
                                      ->  Result  (cost=36.84..45.03 rows=1 width=1275) (actual time=0.067..0.105 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                            Buffers: shared hit=35
                                            ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=1267) (actual time=0.065..0.102 rows=3 loops=1)
                                                  Buffers: shared hit=35
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1283) (actual time=0.055..0.078 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.041..0.044 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.036 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.010..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=10
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.918 ms
Execution Time: 3.906 ms
```

### attempt 1: band (ii) kp=3, agents, with the budgeted count

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=13294.36..13294.37 rows=1 width=8) (actual time=9.798..9.805 rows=1 loops=1)
  Buffers: shared hit=7668
  ->  Limit  (cost=0.98..13281.85 rows=1001 width=4) (actual time=0.041..9.649 rows=1001 loops=1)
        Buffers: shared hit=7668
        ->  Result  (cost=0.98..13693.14 rows=1032 width=4) (actual time=0.040..9.509 rows=1001 loops=1)
              Buffers: shared hit=7668
              ->  Append  (cost=0.98..13682.82 rows=1032 width=0) (actual time=0.040..9.325 rows=1001 loops=1)
                    Buffers: shared hit=7668
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.98..12645.37 rows=1001 width=0) (actual time=0.039..9.168 rows=1001 loops=1)
                          Buffers: shared hit=7668
                          ->  Limit  (cost=0.98..12635.36 rows=1001 width=4) (actual time=0.038..8.999 rows=1001 loops=1)
                                Buffers: shared hit=7668
                                ->  Result  (cost=0.98..18327.76 rows=1452 width=4) (actual time=0.038..8.856 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=7668
                                      ->  Nested Loop Semi Join  (cost=0.98..18327.76 rows=1452 width=0) (actual time=0.033..8.694 rows=1001 loops=1)
                                            Buffers: shared hit=7668
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..537.59 rows=6055 width=32) (actual time=0.023..0.701 rows=2164 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1175
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.93 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=2164)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=6493
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..881.46 rows=27 width=0) (never executed)
                          ->  Limit  (cost=0.86..881.19 rows=27 width=4) (never executed)
                                ->  Result  (cost=0.86..881.19 rows=27 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..881.19 rows=27 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..385.94 rows=112 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.41 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..105.79 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..105.76 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..105.76 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..105.76 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..47.80 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.84..45.04 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.84..45.03 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.84..45.03 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = 'bc317bc2-5d1a-4bda-a773-efa66e89de4c'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.649 ms
Execution Time: 9.915 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk = 3 (broad 50% value, 6% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17800.12..17800.13 rows=1 width=8) (actual time=50.081..54.782 rows=1 loops=1)
  Buffers: shared hit=10677
  ->  Limit  (cost=16836.95..17787.61 rows=1001 width=4) (actual time=48.789..54.724 rows=1001 loops=1)
        Buffers: shared hit=10677
        ->  Result  (cost=16836.95..17794.26 rows=1008 width=4) (actual time=48.788..54.661 rows=1001 loops=1)
              Buffers: shared hit=10677
              ->  Append  (cost=16836.95..17784.18 rows=1008 width=0) (actual time=48.788..54.579 rows=1001 loops=1)
                    Buffers: shared hit=10677
                    ->  Subquery Scan on "*SELECT* 1"  (cost=16836.95..17481.02 rows=1001 width=0) (actual time=48.787..54.513 rows=1001 loops=1)
                          Buffers: shared hit=10677
                          ->  Limit  (cost=16836.95..17471.01 rows=1001 width=4) (actual time=48.786..54.441 rows=1001 loops=1)
                                Buffers: shared hit=10677
                                ->  Gather  (cost=16836.95..17614.17 rows=1227 width=4) (actual time=48.785..54.388 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=10677
                                      ->  Result  (cost=15836.95..16491.47 rows=722 width=4) (actual time=47.374..48.397 rows=708 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                            Buffers: shared hit=10677
                                            ->  Parallel Hash Semi Join  (cost=15836.95..16491.47 rows=722 width=0) (actual time=47.369..48.339 rows=708 loops=2)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=10677
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..638.37 rows=3303 width=32) (actual time=0.037..0.384 rows=1488 loops=2)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1557
                                                  ->  Parallel Hash  (cost=14924.33..14924.33 rows=72964 width=32) (actual time=46.784..46.785 rows=77925 loops=2)
                                                        Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                        Buffers: shared hit=9091
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..14924.33 rows=72964 width=32) (actual time=13.199..33.823 rows=77925 loops=2)
                                                              Filter: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision))
                                                              Rows Removed by Filter: 322075
                                                              Buffers: shared hit=9091
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..144.92 rows=3 width=0) (never executed)
                          ->  Limit  (cost=1.11..144.89 rows=3 width=4) (never executed)
                                ->  Result  (cost=1.11..144.89 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..144.89 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.11 rows=15 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.71 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..119.84 rows=3 width=0) (never executed)
                          ->  Limit  (cost=1.11..119.81 rows=3 width=4) (never executed)
                                ->  Result  (cost=1.11..119.81 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..119.81 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..36.78 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.35 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.34 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.34 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.34 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.93 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_3.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.896 ms
Execution Time: 54.868 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, three broad fields

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10526.55..10526.56 rows=1 width=8) (actual time=72.437..78.166 rows=1 loops=1)
  Buffers: shared hit=60483
  ->  Limit  (cost=8666.49..10522.75 rows=304 width=4) (actual time=44.889..78.064 rows=1001 loops=1)
        Buffers: shared hit=60483
        ->  Result  (cost=8666.49..10522.75 rows=304 width=4) (actual time=44.889..77.970 rows=1001 loops=1)
              Buffers: shared hit=60483
              ->  Append  (cost=8666.49..10519.71 rows=304 width=0) (actual time=44.888..77.858 rows=1001 loops=1)
                    Buffers: shared hit=60483
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10213.86 rows=301 width=0) (actual time=44.886..77.761 rows=1001 loops=1)
                          Buffers: shared hit=60483
                          ->  Limit  (cost=8666.49..10210.85 rows=301 width=4) (actual time=44.885..77.658 rows=1001 loops=1)
                                Buffers: shared hit=60483
                                ->  Gather  (cost=8666.49..10210.85 rows=301 width=4) (actual time=44.884..77.571 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=60483
                                      ->  Result  (cost=7666.48..9180.75 rows=177 width=4) (actual time=43.275..70.422 rows=808 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                            Buffers: shared hit=60483
                                            ->  Nested Loop Semi Join  (cost=7666.48..9180.75 rows=177 width=0) (actual time=43.269..70.320 rows=808 loops=2)
                                                  Buffers: shared hit=60483
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8824.29 rows=410 width=64) (actual time=43.241..61.574 rows=2004 loops=2)
                                                        Buffers: shared hit=44446
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8321.28 rows=824 width=48) (actual time=43.203..48.598 rows=4012 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=12353
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..638.37 rows=3303 width=32) (actual time=0.036..1.821 rows=7934 loops=2)
                                                                    Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=8200
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=42.713..42.714 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=0.008..22.399 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '7436f080-d43f-4663-8035-b30e38c0d7c2'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=8023)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v.record_id) AND (field_id = 'c446e0dc-7130-47d7-814b-2e43b4ad4431'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=32093
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..0.86 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=4009)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v.record_id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=16037
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..147.32 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..147.31 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..147.31 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..147.31 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..146.68 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..144.76 rows=3 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.11 rows=15 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..6.71 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v_4.record_id) AND (field_id = 'c446e0dc-7130-47d7-814b-2e43b4ad4431'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v_5.record_id) AND (field_id = '7436f080-d43f-4663-8035-b30e38c0d7c2'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..122.27 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..122.26 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..122.26 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..122.26 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..121.63 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..119.71 rows=3 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..36.78 rows=12 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..6.91 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v_7.record_id) AND (field_id = 'c446e0dc-7130-47d7-814b-2e43b4ad4431'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v_8.record_id) AND (field_id = '7436f080-d43f-4663-8035-b30e38c0d7c2'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..34.74 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..34.73 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..34.73 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..34.73 rows=1 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.85 rows=1 width=80) (never executed)
                                                  Join Filter: (r_3.id = v_9.record_id)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.23 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_3.id) AND (field_id = 'c446e0dc-7130-47d7-814b-2e43b4ad4431'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                  ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v_11.record_id) AND (field_id = '7436f080-d43f-4663-8035-b30e38c0d7c2'::uuid))
                                                        Filter: value
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.42..0.86 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = v_9.record_id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.760 ms
Execution Time: 78.287 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk = 20 (partly correlated, 500 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=8207.09..8207.10 rows=1 width=8) (actual time=17.819..17.826 rows=1 loops=1)
  Buffers: shared hit=17983
  ->  Limit  (cost=7229.27..8206.05 rows=83 width=4) (actual time=10.952..17.783 rows=494 loops=1)
        Buffers: shared hit=17983
        ->  Result  (cost=7229.27..8206.05 rows=83 width=4) (actual time=10.951..17.742 rows=494 loops=1)
              Buffers: shared hit=17983
              ->  Append  (cost=7229.27..8205.22 rows=83 width=0) (actual time=10.950..17.685 rows=494 loops=1)
                    Buffers: shared hit=17983
                    ->  Subquery Scan on "*SELECT* 1"  (cost=7229.27..7906.73 rows=80 width=0) (actual time=10.950..17.586 rows=494 loops=1)
                          Buffers: shared hit=17969
                          ->  Limit  (cost=7229.27..7905.93 rows=80 width=4) (actual time=10.949..17.540 rows=494 loops=1)
                                Buffers: shared hit=17969
                                ->  Result  (cost=7229.27..7905.93 rows=80 width=4) (actual time=10.948..17.500 rows=494 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=17969
                                      ->  Hash Semi Join  (cost=7229.27..7905.93 rows=80 width=0) (actual time=10.943..17.447 rows=494 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=17969
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..661.49 rows=5615 width=32) (actual time=0.021..4.037 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13085
                                            ->  Hash  (cost=7086.53..7086.53 rows=11374 width=32) (actual time=10.898..10.898 rows=21487 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1599kB
                                                  Buffers: shared hit=4884
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..7086.53 rows=11374 width=32) (actual time=0.018..8.092 rows=21487 loops=1)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '20'::double precision))
                                                        Heap Fetches: 21487
                                                        Buffers: shared hit=4884
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..144.90 rows=1 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..144.89 rows=1 width=4) (actual time=0.021..0.021 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..144.89 rows=1 width=4) (actual time=0.019..0.020 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..144.89 rows=1 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.11 rows=15 width=32) (actual time=0.011..0.012 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.71 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '20'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..119.82 rows=1 width=0) (actual time=0.009..0.011 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..119.81 rows=1 width=4) (actual time=0.009..0.010 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..119.81 rows=1 width=4) (actual time=0.009..0.009 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..119.81 rows=1 width=0) (actual time=0.007..0.008 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..36.78 rows=12 width=32) (actual time=0.007..0.007 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '20'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.35 rows=1 width=0) (actual time=0.018..0.020 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.34 rows=1 width=4) (actual time=0.018..0.019 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.34 rows=1 width=4) (actual time=0.017..0.018 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.34 rows=1 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.015..0.016 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.015..0.016 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.012..0.013 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.006..0.006 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.004..0.004 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.93 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_3.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value = '20'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.062 ms
Execution Time: 17.915 ms
```

### S-A2-cancel candidate, unbudgeted count: a50, pk = 3 (broad 50% value, 50% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9747.12..9747.13 rows=1 width=8) (actual time=6.002..6.007 rows=1 loops=1)
  Buffers: shared hit=10405
  ->  Limit  (cost=1.11..9734.61 rows=1001 width=4) (actual time=0.038..5.936 rows=1001 loops=1)
        Buffers: shared hit=10405
        ->  Result  (cost=1.11..9783.23 rows=1006 width=4) (actual time=0.037..5.867 rows=1001 loops=1)
              Buffers: shared hit=10405
              ->  Append  (cost=1.11..9773.17 rows=1006 width=0) (actual time=0.037..5.771 rows=1001 loops=1)
                    Buffers: shared hit=10405
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1.11..9518.00 rows=1001 width=0) (actual time=0.036..5.694 rows=1001 loops=1)
                          Buffers: shared hit=10405
                          ->  Limit  (cost=1.11..9507.99 rows=1001 width=4) (actual time=0.036..5.610 rows=1001 loops=1)
                                Buffers: shared hit=10405
                                ->  Result  (cost=1.11..98128.01 rows=10332 width=4) (actual time=0.036..5.549 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=10405
                                      ->  Nested Loop Semi Join  (cost=1.11..98128.01 rows=10332 width=0) (actual time=0.032..5.462 rows=1001 loops=1)
                                            Buffers: shared hit=10405
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5383.35 rows=47290 width=32) (actual time=0.021..0.344 rows=2082 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1075
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1.95 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=2082)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 1001
                                                  Buffers: shared hit=9330
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..118.57 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..118.55 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..118.55 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..118.55 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.11 rows=11 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.76 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..98.07 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..98.05 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..98.05 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..98.05 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..36.78 rows=9 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.80 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.49 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.48 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.48 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.48 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,0b7b2d17-8a78-49be-96e4-9df2f001410a}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '74c7a274-01ce-4006-8683-9a25f9d6697e'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.07 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_3.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.990 ms
Execution Time: 6.079 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk < 5 (every 3 and 4: 78% of the side rows)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=15611.47..15611.48 rows=1 width=8) (actual time=3.585..3.590 rows=1 loops=1)
  Buffers: shared hit=4629
  ->  Limit  (cost=0.99..15598.95 rows=1001 width=4) (actual time=0.030..3.524 rows=1001 loops=1)
        Buffers: shared hit=4629
        ->  Result  (cost=0.99..15801.53 rows=1014 width=4) (actual time=0.030..3.458 rows=1001 loops=1)
              Buffers: shared hit=4629
              ->  Append  (cost=0.99..15791.39 rows=1014 width=0) (actual time=0.030..3.374 rows=1001 loops=1)
                    Buffers: shared hit=4629
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.99..15443.52 rows=1001 width=0) (actual time=0.029..3.304 rows=1001 loops=1)
                          Buffers: shared hit=4629
                          ->  Limit  (cost=0.99..15433.51 rows=1001 width=4) (actual time=0.029..3.230 rows=1001 loops=1)
                                Buffers: shared hit=4629
                                ->  Result  (cost=0.99..38420.40 rows=2492 width=4) (actual time=0.029..3.161 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      Buffers: shared hit=4629
                                      ->  Nested Loop Semi Join  (cost=0.99..38420.40 rows=2492 width=0) (actual time=0.026..3.078 rows=1001 loops=1)
                                            Buffers: shared hit=4629
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..661.49 rows=5615 width=32) (actual time=0.016..0.170 rows=1022 loops=1)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=541
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..6.71 rows=1 width=32) (actual time=0.003..0.003 rows=1 loops=1022)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value < '5'::double precision)
                                                  Rows Removed by Filter: 0
                                                  Buffers: shared hit=4088
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..171.09 rows=7 width=0) (never executed)
                          ->  Limit  (cost=0.99..171.02 rows=7 width=4) (never executed)
                                ->  Result  (cost=0.99..171.02 rows=7 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..171.02 rows=7 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..44.11 rows=15 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_1.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..138.36 rows=5 width=0) (never executed)
                          ->  Limit  (cost=0.99..138.31 rows=5 width=4) (never executed)
                                ->  Result  (cost=0.99..138.31 rows=5 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..138.31 rows=5 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..36.78 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_2.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value < '5'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.35 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.34 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.34 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.34 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,ddeba63b-1357-47c8-90d0-1936cb288fbf}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '23e06e1d-57f7-445c-83a5-de69a64cf0f9'::uuid)) AND (entity_definition_id = '01a2003f-0905-7ad6-b5c4-a5245d78f573'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.93 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'faf09730-fbb6-4b07-b56a-65cafa770426'::uuid) AND (record_id = r_3.id) AND (field_id = '58d9aaf3-f93f-4564-bd06-34ab483efe53'::uuid))
                                                  Filter: (value < '5'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.784 ms
Execution Time: 3.647 ms
```
