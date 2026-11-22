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
Nested Loop Left Join  (cost=48.24..10699.59 rows=1 width=1317) (actual time=0.389..3.040 rows=51 loops=1)
  Buffers: shared hit=2253
  ->  Nested Loop  (cost=47.94..10691.26 rows=1 width=1282) (actual time=0.373..2.899 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2100
        ->  Subquery Scan on x  (cost=47.66..10682.94 rows=1 width=1275) (actual time=0.361..2.795 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1947
              ->  Limit  (cost=47.66..10681.80 rows=51 width=1275) (actual time=0.360..2.758 rows=51 loops=1)
                    Buffers: shared hit=1947
                    ->  Merge Append  (cost=47.66..17771.22 rows=85 width=1275) (actual time=0.359..2.750 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1947
                          ->  Limit  (cost=0.86..16580.48 rows=51 width=1275) (actual time=0.191..2.565 rows=51 loops=1)
                                Buffers: shared hit=1892
                                ->  Result  (cost=0.86..457728.42 rows=1408 width=1275) (actual time=0.191..2.558 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=1892
                                      ->  Nested Loop Semi Join  (cost=0.86..457728.42 rows=1408 width=1267) (actual time=0.187..2.540 rows=51 loops=1)
                                            Buffers: shared hit=1892
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..440269.73 rows=5890 width=1267) (actual time=0.086..2.091 rows=100 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1470
                                                  Buffers: shared hit=1591
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.95 rows=1 width=32) (actual time=0.004..0.004 rows=1 loops=100)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=301
                          ->  Limit  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.037..0.038 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.037..0.038 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=12
                                      ->  Nested Loop Semi Join  (cost=0.86..1033.32 rows=30 width=1267) (actual time=0.035..0.036 rows=1 loops=1)
                                            Buffers: shared hit=12
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.58 rows=124 width=1267) (actual time=0.017..0.019 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=7
                          ->  Limit  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.026..0.028 rows=1 loops=1)
                                Buffers: shared hit=8
                                ->  Result  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.025..0.027 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=8
                                      ->  Nested Loop Semi Join  (cost=0.86..111.06 rows=3 width=1267) (actual time=0.024..0.025 rows=1 loops=1)
                                            Buffers: shared hit=8
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.10 rows=13 width=1267) (actual time=0.014..0.014 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=4
                          ->  Limit  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.103..0.107 rows=1 loops=1)
                                Buffers: shared hit=35
                                ->  Sort  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.103..0.106 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=35
                                      ->  Result  (cost=36.84..45.03 rows=1 width=1275) (actual time=0.067..0.098 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                            Buffers: shared hit=35
                                            ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=1267) (actual time=0.065..0.095 rows=3 loops=1)
                                                  Buffers: shared hit=35
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1283) (actual time=0.053..0.072 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.040 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.035 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=10
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.824 ms
Execution Time: 3.201 ms
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
Nested Loop Left Join  (cost=48.24..10699.59 rows=1 width=1317) (actual time=0.837..5.837 rows=51 loops=1)
  Buffers: shared hit=2253
  ->  Nested Loop  (cost=47.94..10691.26 rows=1 width=1282) (actual time=0.802..5.522 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2100
        ->  Subquery Scan on x  (cost=47.66..10682.94 rows=1 width=1275) (actual time=0.777..5.293 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1947
              ->  Limit  (cost=47.66..10681.80 rows=51 width=1275) (actual time=0.773..5.208 rows=51 loops=1)
                    Buffers: shared hit=1947
                    ->  Merge Append  (cost=47.66..17771.22 rows=85 width=1275) (actual time=0.770..5.191 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1947
                          ->  Limit  (cost=0.86..16580.48 rows=51 width=1275) (actual time=0.413..4.801 rows=51 loops=1)
                                Buffers: shared hit=1892
                                ->  Result  (cost=0.86..457728.42 rows=1408 width=1275) (actual time=0.412..4.786 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=1892
                                      ->  Nested Loop Semi Join  (cost=0.86..457728.42 rows=1408 width=1267) (actual time=0.403..4.749 rows=51 loops=1)
                                            Buffers: shared hit=1892
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..440269.73 rows=5890 width=1267) (actual time=0.213..3.855 rows=100 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1470
                                                  Buffers: shared hit=1591
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.95 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=100)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=301
                          ->  Limit  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.075..0.078 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.074..0.076 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=12
                                      ->  Nested Loop Semi Join  (cost=0.86..1033.32 rows=30 width=1267) (actual time=0.068..0.069 rows=1 loops=1)
                                            Buffers: shared hit=12
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.58 rows=124 width=1267) (actual time=0.031..0.035 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (actual time=0.015..0.015 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=7
                          ->  Limit  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.062..0.064 rows=1 loops=1)
                                Buffers: shared hit=8
                                ->  Result  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.061..0.063 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=8
                                      ->  Nested Loop Semi Join  (cost=0.86..111.06 rows=3 width=1267) (actual time=0.058..0.059 rows=1 loops=1)
                                            Buffers: shared hit=8
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.10 rows=13 width=1267) (actual time=0.033..0.034 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (actual time=0.021..0.021 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=4
                          ->  Limit  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.216..0.221 rows=1 loops=1)
                                Buffers: shared hit=35
                                ->  Sort  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.216..0.220 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=35
                                      ->  Result  (cost=36.84..45.03 rows=1 width=1275) (actual time=0.141..0.201 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                            Buffers: shared hit=35
                                            ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=1267) (actual time=0.137..0.195 rows=3 loops=1)
                                                  Buffers: shared hit=35
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1283) (actual time=0.111..0.147 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.081..0.085 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.058..0.071 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.047..0.048 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.025..0.025 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.018..0.018 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=10
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.196 ms
Execution Time: 6.213 ms
```

### attempt 1: band (ii) kp=3, agents, with the budgeted count

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=13548.25..13548.26 rows=1 width=8) (actual time=12.624..12.639 rows=1 loops=1)
  Buffers: shared hit=7668
  ->  Limit  (cost=0.98..13535.74 rows=1001 width=4) (actual time=0.047..12.466 rows=1001 loops=1)
        Buffers: shared hit=7668
        ->  Result  (cost=0.98..13995.46 rows=1035 width=4) (actual time=0.046..12.297 rows=1001 loops=1)
              Buffers: shared hit=7668
              ->  Append  (cost=0.98..13985.11 rows=1035 width=0) (actual time=0.046..12.072 rows=1001 loops=1)
                    Buffers: shared hit=7668
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.98..12790.18 rows=1001 width=0) (actual time=0.045..11.876 rows=1001 loops=1)
                          Buffers: shared hit=7668
                          ->  Limit  (cost=0.98..12780.17 rows=1001 width=4) (actual time=0.045..11.675 rows=1001 loops=1)
                                Buffers: shared hit=7668
                                ->  Result  (cost=0.98..17976.11 rows=1408 width=4) (actual time=0.045..11.526 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=7668
                                      ->  Nested Loop Semi Join  (cost=0.98..17976.11 rows=1408 width=0) (actual time=0.039..11.326 rows=1001 loops=1)
                                            Buffers: shared hit=7668
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.42 rows=5890 width=32) (actual time=0.026..0.863 rows=2164 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1175
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.95 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=2164)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=6493
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1033.62 rows=30 width=0) (never executed)
                          ->  Limit  (cost=0.86..1033.32 rows=30 width=4) (never executed)
                                ->  Result  (cost=0.86..1033.32 rows=30 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1033.32 rows=30 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.58 rows=124 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..111.09 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..111.06 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..111.06 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..111.06 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.10 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.84..45.04 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.84..45.03 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.84..45.03 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.442 ms
Execution Time: 12.783 ms
```

### attempt 2: band (ii) kp=3, agents, count skipped

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
Nested Loop Left Join  (cost=48.24..10699.59 rows=1 width=1317) (actual time=0.961..6.397 rows=51 loops=1)
  Buffers: shared hit=2253
  ->  Nested Loop  (cost=47.94..10691.26 rows=1 width=1282) (actual time=0.924..6.050 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2100
        ->  Subquery Scan on x  (cost=47.66..10682.94 rows=1 width=1275) (actual time=0.894..5.795 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1947
              ->  Limit  (cost=47.66..10681.80 rows=51 width=1275) (actual time=0.888..5.696 rows=51 loops=1)
                    Buffers: shared hit=1947
                    ->  Merge Append  (cost=47.66..17771.22 rows=85 width=1275) (actual time=0.884..5.677 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1947
                          ->  Limit  (cost=0.86..16580.48 rows=51 width=1275) (actual time=0.472..5.225 rows=51 loops=1)
                                Buffers: shared hit=1892
                                ->  Result  (cost=0.86..457728.42 rows=1408 width=1275) (actual time=0.471..5.208 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=1892
                                      ->  Nested Loop Semi Join  (cost=0.86..457728.42 rows=1408 width=1267) (actual time=0.457..5.163 rows=51 loops=1)
                                            Buffers: shared hit=1892
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..440269.73 rows=5890 width=1267) (actual time=0.167..4.057 rows=100 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1470
                                                  Buffers: shared hit=1591
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.95 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=100)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=301
                          ->  Limit  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.099..0.102 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.097..0.100 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=12
                                      ->  Nested Loop Semi Join  (cost=0.86..1033.32 rows=30 width=1267) (actual time=0.086..0.087 rows=1 loops=1)
                                            Buffers: shared hit=12
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.58 rows=124 width=1267) (actual time=0.040..0.045 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=7
                          ->  Limit  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.068..0.071 rows=1 loops=1)
                                Buffers: shared hit=8
                                ->  Result  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.067..0.070 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=8
                                      ->  Nested Loop Semi Join  (cost=0.86..111.06 rows=3 width=1267) (actual time=0.061..0.063 rows=1 loops=1)
                                            Buffers: shared hit=8
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.10 rows=13 width=1267) (actual time=0.034..0.035 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (actual time=0.023..0.024 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=4
                          ->  Limit  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.239..0.247 rows=1 loops=1)
                                Buffers: shared hit=35
                                ->  Sort  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.238..0.246 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=35
                                      ->  Result  (cost=36.84..45.03 rows=1 width=1275) (actual time=0.153..0.221 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                            Buffers: shared hit=35
                                            ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=1267) (actual time=0.148..0.214 rows=3 loops=1)
                                                  Buffers: shared hit=35
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1283) (actual time=0.121..0.161 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.087..0.091 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.062..0.075 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.050..0.052 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.025..0.025 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.020..0.020 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=10
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.311 ms
Execution Time: 6.811 ms
```

### attempt 2: band (ii) kp=3, agents, with the budgeted count

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
Nested Loop Left Join  (cost=48.24..10699.59 rows=1 width=1317) (actual time=1.131..9.162 rows=51 loops=1)
  Buffers: shared hit=2253
  ->  Nested Loop  (cost=47.94..10691.26 rows=1 width=1282) (actual time=1.087..8.486 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2100
        ->  Subquery Scan on x  (cost=47.66..10682.94 rows=1 width=1275) (actual time=1.056..8.076 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1947
              ->  Limit  (cost=47.66..10681.80 rows=51 width=1275) (actual time=1.039..7.912 rows=51 loops=1)
                    Buffers: shared hit=1947
                    ->  Merge Append  (cost=47.66..17771.22 rows=85 width=1275) (actual time=1.036..7.887 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1947
                          ->  Limit  (cost=0.86..16580.48 rows=51 width=1275) (actual time=0.392..7.131 rows=51 loops=1)
                                Buffers: shared hit=1892
                                ->  Result  (cost=0.86..457728.42 rows=1408 width=1275) (actual time=0.391..7.108 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=1892
                                      ->  Nested Loop Semi Join  (cost=0.86..457728.42 rows=1408 width=1267) (actual time=0.377..7.047 rows=51 loops=1)
                                            Buffers: shared hit=1892
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..440269.73 rows=5890 width=1267) (actual time=0.153..5.589 rows=100 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1470
                                                  Buffers: shared hit=1591
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.95 rows=1 width=32) (actual time=0.013..0.013 rows=1 loops=100)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=301
                          ->  Limit  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.081..0.100 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Result  (cost=0.86..1033.32 rows=30 width=1275) (actual time=0.080..0.097 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=12
                                      ->  Nested Loop Semi Join  (cost=0.86..1033.32 rows=30 width=1267) (actual time=0.074..0.086 rows=1 loops=1)
                                            Buffers: shared hit=12
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.58 rows=124 width=1267) (actual time=0.033..0.040 rows=2 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=5
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (actual time=0.016..0.017 rows=0 loops=2)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=7
                          ->  Limit  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.078..0.093 rows=1 loops=1)
                                Buffers: shared hit=8
                                ->  Result  (cost=0.86..111.06 rows=3 width=1275) (actual time=0.077..0.087 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=8
                                      ->  Nested Loop Semi Join  (cost=0.86..111.06 rows=3 width=1267) (actual time=0.073..0.079 rows=1 loops=1)
                                            Buffers: shared hit=8
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.10 rows=13 width=1267) (actual time=0.038..0.039 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (actual time=0.028..0.028 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=4
                          ->  Limit  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.478..0.508 rows=1 loops=1)
                                Buffers: shared hit=35
                                ->  Sort  (cost=45.04..45.05 rows=1 width=1275) (actual time=0.476..0.503 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=35
                                      ->  Result  (cost=36.84..45.03 rows=1 width=1275) (actual time=0.370..0.469 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                            Buffers: shared hit=35
                                            ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=1267) (actual time=0.362..0.456 rows=3 loops=1)
                                                  Buffers: shared hit=35
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1283) (actual time=0.319..0.376 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.246..0.260 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.212..0.236 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.185..0.192 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.027..0.028 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.156..0.159 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1267) (actual time=0.033..0.033 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (actual time=0.021..0.022 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=10
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.934 ms
Execution Time: 9.579 ms
```

### attempt 2: band (ii) kp=3, agents, with the budgeted count

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=13548.25..13548.26 rows=1 width=8) (actual time=29.104..29.119 rows=1 loops=1)
  Buffers: shared hit=7668
  ->  Limit  (cost=0.98..13535.74 rows=1001 width=4) (actual time=0.117..28.677 rows=1001 loops=1)
        Buffers: shared hit=7668
        ->  Result  (cost=0.98..13995.46 rows=1035 width=4) (actual time=0.115..28.385 rows=1001 loops=1)
              Buffers: shared hit=7668
              ->  Append  (cost=0.98..13985.11 rows=1035 width=0) (actual time=0.114..28.004 rows=1001 loops=1)
                    Buffers: shared hit=7668
                    ->  Subquery Scan on "*SELECT* 1"  (cost=0.98..12790.18 rows=1001 width=0) (actual time=0.113..27.584 rows=1001 loops=1)
                          Buffers: shared hit=7668
                          ->  Limit  (cost=0.98..12780.17 rows=1001 width=4) (actual time=0.111..27.212 rows=1001 loops=1)
                                Buffers: shared hit=7668
                                ->  Result  (cost=0.98..17976.11 rows=1408 width=4) (actual time=0.111..26.936 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=7668
                                      ->  Nested Loop Semi Join  (cost=0.98..17976.11 rows=1408 width=0) (actual time=0.090..26.432 rows=1001 loops=1)
                                            Buffers: shared hit=7668
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..517.42 rows=5890 width=32) (actual time=0.059..2.419 rows=2164 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1175
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..2.95 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=2164)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=6493
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1033.62 rows=30 width=0) (never executed)
                          ->  Limit  (cost=0.86..1033.32 rows=30 width=4) (never executed)
                                ->  Result  (cost=0.86..1033.32 rows=30 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1033.32 rows=30 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..484.58 rows=124 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..4.42 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..111.09 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..111.06 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..111.06 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..111.06 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..53.10 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..4.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.84..45.04 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.84..45.03 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.84..45.03 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.84..45.03 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_3  (cost=0.42..0.56 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9349c71-e0f4-4a8c-99a3-40058facb9de'::uuid) AND (value = '3'::double precision) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.550 ms
Execution Time: 29.492 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk = 3 (broad 50% value, 6% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17818.50..17818.51 rows=1 width=8) (actual time=489.672..516.659 rows=1 loops=1)
  Buffers: shared hit=10549
  ->  Limit  (cost=16827.62..17805.99 rows=1001 width=4) (actual time=484.556..516.255 rows=1001 loops=1)
        Buffers: shared hit=10549
        ->  Result  (cost=16827.62..17812.83 rows=1008 width=4) (actual time=484.552..516.046 rows=1001 loops=1)
              Buffers: shared hit=10549
              ->  Append  (cost=16827.62..17802.75 rows=1008 width=0) (actual time=484.550..515.714 rows=1001 loops=1)
                    Buffers: shared hit=10549
                    ->  Subquery Scan on "*SELECT* 1"  (cost=16827.62..17474.25 rows=1001 width=0) (actual time=484.548..515.500 rows=1001 loops=1)
                          Buffers: shared hit=10549
                          ->  Limit  (cost=16827.62..17464.24 rows=1001 width=4) (actual time=484.545..515.252 rows=1001 loops=1)
                                Buffers: shared hit=10549
                                ->  Gather  (cost=16827.62..17665.84 rows=1318 width=4) (actual time=484.543..515.000 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=10549
                                      ->  Result  (cost=15827.62..16534.04 rows=775 width=4) (actual time=421.031..425.017 rows=650 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                            Buffers: shared hit=10549
                                            ->  Parallel Hash Semi Join  (cost=15827.62..16534.04 rows=775 width=0) (actual time=421.006..424.827 rows=650 loops=2)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=10549
                                                  ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.90 rows=3600 width=32) (actual time=0.249..1.535 rows=1362 loops=2)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1429
                                                  ->  Parallel Hash  (cost=14924.33..14924.33 rows=72218 width=32) (actual time=417.970..417.973 rows=77925 loops=2)
                                                        Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                        Buffers: shared hit=9091
                                                        ->  Parallel Seq Scan on rfv_num v  (cost=0.00..14924.33 rows=72218 width=32) (actual time=103.497..293.314 rows=77925 loops=2)
                                                              Filter: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision))
                                                              Rows Removed by Filter: 322075
                                                              Buffers: shared hit=9091
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..156.25 rows=3 width=0) (never executed)
                          ->  Limit  (cost=1.11..156.22 rows=3 width=4) (never executed)
                                ->  Result  (cost=1.11..156.22 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..156.22 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..55.43 rows=15 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.71 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..133.89 rows=3 width=0) (never executed)
                          ->  Limit  (cost=1.11..133.86 rows=3 width=4) (never executed)
                                ->  Result  (cost=1.11..133.86 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..133.86 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..46.24 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.73 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.33 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.32 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.32 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.32 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = r_3.id) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.615 ms
Execution Time: 517.206 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, three broad fields

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value = $11::double precision) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $12::uuid AND v.value = $13::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=10680.92..10680.93 rows=1 width=8) (actual time=417.863..484.211 rows=1 loops=1)
  Buffers: shared hit=59982
  ->  Limit  (cost=8666.49..10676.86 rows=325 width=4) (actual time=230.264..481.699 rows=1001 loops=1)
        Buffers: shared hit=59982
        ->  Result  (cost=8666.49..10676.86 rows=325 width=4) (actual time=230.260..481.095 rows=1001 loops=1)
              Buffers: shared hit=59982
              ->  Append  (cost=8666.49..10673.61 rows=325 width=0) (actual time=230.258..480.602 rows=1001 loops=1)
                    Buffers: shared hit=59982
                    ->  Subquery Scan on "*SELECT* 1"  (cost=8666.49..10342.30 rows=322 width=0) (actual time=230.255..480.075 rows=1001 loops=1)
                          Buffers: shared hit=59982
                          ->  Limit  (cost=8666.49..10339.08 rows=322 width=4) (actual time=230.253..479.365 rows=1001 loops=1)
                                Buffers: shared hit=59982
                                ->  Gather  (cost=8666.49..10339.08 rows=322 width=4) (actual time=230.252..478.912 rows=1001 loops=1)
                                      Workers Planned: 1
                                      Workers Launched: 1
                                      Buffers: shared hit=59982
                                      ->  Result  (cost=7666.48..9306.88 rows=189 width=4) (actual time=224.718..408.681 rows=802 loops=2)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                            Buffers: shared hit=59982
                                            ->  Nested Loop Semi Join  (cost=7666.48..9306.88 rows=189 width=0) (actual time=224.698..408.103 rows=802 loops=2)
                                                  Buffers: shared hit=59982
                                                  ->  Nested Loop Semi Join  (cost=7666.05..8919.12 rows=446 width=64) (actual time=224.480..347.537 rows=1988 loops=2)
                                                        Buffers: shared hit=44081
                                                        ->  Parallel Hash Semi Join  (cost=7665.62..8373.37 rows=894 width=48) (actual time=224.393..255.852 rows=3974 loops=2)
                                                              Hash Cond: (r.id = v.record_id)
                                                              Buffers: shared hit=12292
                                                              ->  Parallel Index Only Scan using records_list_group on records r  (cost=0.55..688.90 rows=3600 width=32) (actual time=0.126..10.700 rows=7860 loops=2)
                                                                    Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=8139
                                                              ->  Parallel Hash  (cost=6624.00..6624.00 rows=83286 width=32) (actual time=222.655..222.657 rows=99944 loops=2)
                                                                    Buckets: 262144  Batches: 1  Memory Usage: 0kB
                                                                    Buffers: shared hit=4124
                                                                    ->  Parallel Seq Scan on rfv_bool v  (cost=0.00..6624.00 rows=83286 width=32) (actual time=0.053..106.406 rows=99944 loops=2)
                                                                          Filter: (value AND (tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd3bbc226-30d5-4c39-9000-c502060ae32d'::uuid))
                                                                          Rows Removed by Filter: 100056
                                                                          Buffers: shared hit=4124
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.42..0.61 rows=1 width=32) (actual time=0.022..0.022 rows=1 loops=7947)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v.record_id) AND (field_id = '8f61290f-509f-46a8-a6aa-94b2e3e8c7f8'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                              Rows Removed by Filter: 0
                                                              Buffers: shared hit=31789
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..0.86 rows=1 width=32) (actual time=0.029..0.029 rows=0 loops=3975)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v.record_id) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid))
                                                        Filter: (value = '3'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=15901
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.96..158.65 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..158.64 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..158.64 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..158.64 rows=1 width=0) (never executed)
                                            Join Filter: (r_1.id = v_3.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..158.02 rows=1 width=64) (never executed)
                                                  Join Filter: (r_1.id = v_5.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..156.09 rows=3 width=48) (never executed)
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.55..55.43 rows=15 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_4  (cost=0.55..6.71 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_5  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v_4.record_id) AND (field_id = '8f61290f-509f-46a8-a6aa-94b2e3e8c7f8'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_3  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'd3bbc226-30d5-4c39-9000-c502060ae32d'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.96..136.31 rows=1 width=0) (never executed)
                          ->  Limit  (cost=1.96..136.30 rows=1 width=4) (never executed)
                                ->  Result  (cost=1.96..136.30 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.96..136.30 rows=1 width=0) (never executed)
                                            Join Filter: (r_2.id = v_6.record_id)
                                            ->  Nested Loop Semi Join  (cost=1.52..135.67 rows=1 width=64) (never executed)
                                                  Join Filter: (r_2.id = v_8.record_id)
                                                  ->  Nested Loop Semi Join  (cost=1.10..133.74 rows=3 width=48) (never executed)
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..46.24 rows=13 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                              Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)))
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_7  (cost=0.55..6.73 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                              Heap Fetches: 0
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_8  (cost=0.42..0.63 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v_7.record_id) AND (field_id = '8f61290f-509f-46a8-a6aa-94b2e3e8c7f8'::uuid))
                                                        Filter: (value = 'x'::text COLLATE "C")
                                            ->  Index Scan using rfv_bool_pkey on rfv_bool v_6  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v_8.record_id) AND (field_id = 'd3bbc226-30d5-4c39-9000-c502060ae32d'::uuid))
                                                  Filter: value
                    ->  Subquery Scan on "*SELECT* 4"  (cost=25.63..34.72 rows=1 width=0) (never executed)
                          ->  Limit  (cost=25.63..34.71 rows=1 width=4) (never executed)
                                ->  Result  (cost=25.63..34.71 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=25.63..34.71 rows=1 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=25.19..33.83 rows=1 width=80) (never executed)
                                                  Join Filter: (r_3.id = v_9.record_id)
                                                  ->  Nested Loop Semi Join  (cost=24.77..33.21 rows=1 width=64) (never executed)
                                                        ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                              ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                                          Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))))
                                                                          ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                                      Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                                    Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                                        ->  Index Scan using rfv_text_pkey on rfv_text v_11  (cost=0.42..0.62 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = r_3.id) AND (field_id = '8f61290f-509f-46a8-a6aa-94b2e3e8c7f8'::uuid))
                                                              Filter: (value = 'x'::text COLLATE "C")
                                                  ->  Index Scan using rfv_bool_pkey on rfv_bool v_9  (cost=0.42..0.60 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v_11.record_id) AND (field_id = 'd3bbc226-30d5-4c39-9000-c502060ae32d'::uuid))
                                                        Filter: value
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_10  (cost=0.42..0.86 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = v_9.record_id) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 15.800 ms
Execution Time: 485.257 ms
```

### S-A2-cancel candidate, unbudgeted count: a6, pk = 20 (partly correlated, 500 visible)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=8170.83..8170.84 rows=1 width=8) (actual time=107.685..107.703 rows=1 loops=1)
  Buffers: shared hit=17995
  ->  Limit  (cost=7113.52..8169.74 rows=87 width=4) (actual time=55.342..107.189 rows=494 loops=1)
        Buffers: shared hit=17995
        ->  Result  (cost=7113.52..8169.74 rows=87 width=4) (actual time=55.340..107.019 rows=494 loops=1)
              Buffers: shared hit=17995
              ->  Append  (cost=7113.52..8168.87 rows=87 width=0) (actual time=55.338..106.732 rows=494 loops=1)
                    Buffers: shared hit=17995
                    ->  Subquery Scan on "*SELECT* 1"  (cost=7113.52..7845.01 rows=84 width=0) (actual time=55.336..106.347 rows=494 loops=1)
                          Buffers: shared hit=17981
                          ->  Limit  (cost=7113.52..7844.17 rows=84 width=4) (actual time=55.333..106.006 rows=494 loops=1)
                                Buffers: shared hit=17981
                                ->  Result  (cost=7113.52..7844.17 rows=84 width=4) (actual time=55.331..105.845 rows=494 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=17981
                                      ->  Hash Semi Join  (cost=7113.52..7844.17 rows=84 width=0) (actual time=55.310..105.581 rows=494 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=17981
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..714.10 rows=6120 width=32) (actual time=0.086..26.897 rows=24055 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13097
                                            ->  Hash  (cost=6974.31..6974.31 rows=11092 width=32) (actual time=55.157..55.158 rows=21487 loops=1)
                                                  Buckets: 32768 (originally 16384)  Batches: 1 (originally 1)  Memory Usage: 1599kB
                                                  Buffers: shared hit=4884
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..6974.31 rows=11092 width=32) (actual time=0.039..39.036 rows=21487 loops=1)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '20'::double precision))
                                                        Heap Fetches: 21487
                                                        Buffers: shared hit=4884
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..156.23 rows=1 width=0) (actual time=0.065..0.068 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..156.22 rows=1 width=4) (actual time=0.065..0.066 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..156.22 rows=1 width=4) (actual time=0.061..0.062 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..156.22 rows=1 width=0) (actual time=0.041..0.042 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..55.43 rows=15 width=32) (actual time=0.039..0.040 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.71 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '20'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..133.87 rows=1 width=0) (actual time=0.030..0.033 rows=0 loops=1)
                          Buffers: shared hit=4
                          ->  Limit  (cost=1.11..133.86 rows=1 width=4) (actual time=0.028..0.030 rows=0 loops=1)
                                Buffers: shared hit=4
                                ->  Result  (cost=1.11..133.86 rows=1 width=4) (actual time=0.027..0.028 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=4
                                      ->  Nested Loop Semi Join  (cost=1.11..133.86 rows=1 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                            Buffers: shared hit=4
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..46.24 rows=13 width=32) (actual time=0.022..0.022 rows=0 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.73 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '20'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.33 rows=1 width=0) (actual time=0.042..0.046 rows=0 loops=1)
                          Buffers: shared hit=6
                          ->  Limit  (cost=24.78..33.32 rows=1 width=4) (actual time=0.041..0.045 rows=0 loops=1)
                                Buffers: shared hit=6
                                ->  Result  (cost=24.78..33.32 rows=1 width=4) (actual time=0.040..0.044 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=6
                                      ->  Nested Loop Semi Join  (cost=24.78..33.32 rows=1 width=0) (actual time=0.037..0.041 rows=0 loops=1)
                                            Buffers: shared hit=6
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (actual time=0.036..0.039 rows=0 loops=1)
                                                  Buffers: shared hit=6
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (actual time=0.036..0.038 rows=0 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=6
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (actual time=0.031..0.033 rows=0 loops=1)
                                                              Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))))
                                                              Buffers: shared hit=6
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=6
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[])))
                                                                          Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,72938a9d-2d94-424b-b195-6b91e585306d}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '9e55883e-2dde-4aba-86b6-9a1dc3419302'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = r_3.id) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid))
                                                  Filter: (value = '20'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.047 ms
Execution Time: 108.035 ms
```

### S-A2-cancel candidate, unbudgeted count: a50, pk = 3 (broad 50% value, 50% ACL)

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT ((r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=9579.83..9579.84 rows=1 width=8) (actual time=38.431..38.447 rows=1 loops=1)
  Buffers: shared hit=10407
  ->  Limit  (cost=1.11..9567.32 rows=1001 width=4) (actual time=0.126..37.900 rows=1001 loops=1)
        Buffers: shared hit=10407
        ->  Result  (cost=1.11..9624.66 rows=1007 width=4) (actual time=0.124..35.850 rows=1001 loops=1)
              Buffers: shared hit=10407
              ->  Append  (cost=1.11..9614.59 rows=1007 width=0) (actual time=0.123..34.765 rows=1001 loops=1)
                    Buffers: shared hit=10407
                    ->  Subquery Scan on "*SELECT* 1"  (cost=1.11..9321.48 rows=1001 width=0) (actual time=0.122..33.442 rows=1001 loops=1)
                          Buffers: shared hit=10407
                          ->  Limit  (cost=1.11..9311.47 rows=1001 width=4) (actual time=0.120..33.004 rows=1001 loops=1)
                                Buffers: shared hit=10407
                                ->  Result  (cost=1.11..100136.23 rows=10766 width=4) (actual time=0.120..32.643 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      Buffers: shared hit=10407
                                      ->  Nested Loop Semi Join  (cost=1.11..100136.23 rows=10766 width=0) (actual time=0.103..32.167 rows=1001 loops=1)
                                            Buffers: shared hit=10407
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..5666.90 rows=49992 width=32) (actual time=0.067..1.959 rows=2082 loops=1)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1077
                                            ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.55..1.88 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=2082)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r.id))
                                                  Heap Fetches: 1001
                                                  Buffers: shared hit=9330
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.11..138.50 rows=3 width=0) (never executed)
                          ->  Limit  (cost=1.11..138.47 rows=3 width=4) (never executed)
                                ->  Result  (cost=1.11..138.47 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..138.47 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.55..55.43 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[])))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.55..6.91 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.11..116.12 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.11..116.10 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.11..116.10 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.11..116.10 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.55..46.24 rows=10 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid))
                                                  Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid)))
                                            ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.55..6.97 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid) AND (value = '3'::double precision) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=24.78..33.46 rows=1 width=0) (never executed)
                          ->  Limit  (cost=24.78..33.45 rows=1 width=4) (never executed)
                                ->  Result  (cost=24.78..33.45 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid)
                                      ->  Nested Loop Semi Join  (cost=24.78..33.45 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=24.35..32.39 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=23.92..23.93 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=12.89..23.92 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid)) OR ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[]))))
                                                              ->  BitmapOr  (cost=12.89..12.89 rows=3 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..8.59 rows=2 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,82d19e8c-19ac-4b3e-9995-868f0ac1f170}'::uuid[]))) AND ((requester_id IS NULL) OR (requester_id <> 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> 'a9f630a8-0902-4e17-8734-ac1002abb04c'::uuid)) AND (entity_definition_id = '01a1fe25-50b7-7d84-a057-045bdd6cbcfc'::uuid))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..1.04 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'd91dc445-cc03-4d75-9fa1-cd48d97b617a'::uuid) AND (record_id = r_3.id) AND (field_id = 'd9567f9f-3fbd-4503-8ece-e854307d1424'::uuid))
                                                  Filter: (value = '3'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.014 ms
Execution Time: 38.804 ms
```
