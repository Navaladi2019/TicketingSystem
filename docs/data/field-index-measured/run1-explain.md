# TICK-747 list plans (one run per cell, as nava_app)

### text eq 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=11443.38..11443.39 rows=1 width=8) (actual time=30.788..30.799 rows=1 loops=1)
  Buffers: shared hit=15886
  ->  Limit  (cost=987.35..11430.87 rows=1001 width=4) (actual time=22.766..30.655 rows=1001 loops=1)
        Buffers: shared hit=15886
        ->  Result  (cost=987.35..11743.86 rows=1031 width=4) (actual time=22.764..30.491 rows=1001 loops=1)
              Buffers: shared hit=15886
              ->  Append  (cost=987.35..11733.55 rows=1031 width=0) (actual time=22.763..30.298 rows=1001 loops=1)
                    Buffers: shared hit=15886
                    ->  Subquery Scan on "*SELECT* 1"  (cost=987.35..10649.35 rows=1001 width=0) (actual time=22.762..30.161 rows=1001 loops=1)
                          Buffers: shared hit=15886
                          ->  Limit  (cost=987.35..10639.34 rows=1001 width=4) (actual time=22.761..30.008 rows=1001 loops=1)
                                Buffers: shared hit=15886
                                ->  Result  (cost=987.35..12905.29 rows=1236 width=4) (actual time=22.759..29.882 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15886
                                      ->  Merge Semi Join  (cost=987.35..12905.29 rows=1236 width=0) (actual time=22.747..29.688 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15886
                                            ->  Sort  (cost=927.09..942.47 rows=6151 width=32) (actual time=22.673..23.493 rows=2496 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: quicksort  Memory: 2087kB
                                                  Buffers: shared hit=15681
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.031..11.043 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..11533.36 rows=160747 width=32) (actual time=0.047..3.848 rows=17379 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=205
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..936.96 rows=26 width=0) (never executed)
                          ->  Limit  (cost=0.99..936.70 rows=26 width=4) (never executed)
                                ->  Result  (cost=0.99..936.70 rows=26 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..936.70 rows=26 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..96.58 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.99..96.55 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.99..96.55 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..96.55 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.50 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.97..45.49 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.97..45.49 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.405 ms
Execution Time: 31.350 ms
```

### text eq 40%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=49.09..9527.08 rows=1 width=1434) (actual time=0.742..6.307 rows=51 loops=1)
  Buffers: shared hit=2960
  ->  Nested Loop  (cost=48.79..9518.75 rows=1 width=1399) (actual time=0.659..5.963 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2807
        ->  Subquery Scan on x  (cost=48.51..9510.44 rows=1 width=1392) (actual time=0.636..5.768 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2654
              ->  Limit  (cost=48.51..9509.29 rows=51 width=1392) (actual time=0.632..5.686 rows=51 loops=1)
                    Buffers: shared hit=2654
                    ->  Merge Append  (cost=48.51..15074.45 rows=81 width=1392) (actual time=0.630..5.671 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2654
                          ->  Limit  (cost=0.99..13994.44 rows=51 width=1392) (actual time=0.266..5.223 rows=50 loops=1)
                                Buffers: shared hit=2578
                                ->  Result  (cost=0.99..339136.36 rows=1236 width=1392) (actual time=0.265..5.210 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2578
                                      ->  Nested Loop Semi Join  (cost=0.99..339136.36 rows=1236 width=1384) (actual time=0.254..5.162 rows=50 loops=1)
                                            Buffers: shared hit=2578
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..314238.64 rows=6151 width=1384) (actual time=0.199..3.902 rows=133 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1887
                                                  Buffers: shared hit=2045
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..4.04 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=133)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=533
                          ->  Limit  (cost=0.99..936.70 rows=26 width=1392) (actual time=0.074..0.126 rows=2 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.99..936.70 rows=26 width=1392) (actual time=0.073..0.124 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=29
                                      ->  Nested Loop Semi Join  (cost=0.99..936.70 rows=26 width=1384) (actual time=0.068..0.116 rows=2 loops=1)
                                            Buffers: shared hit=29
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.037..0.048 rows=5 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=8
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=5)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=21
                          ->  Limit  (cost=0.99..96.55 rows=3 width=1392) (actual time=0.063..0.065 rows=1 loops=1)
                                Buffers: shared hit=9
                                ->  Result  (cost=0.99..96.55 rows=3 width=1392) (actual time=0.062..0.064 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=9
                                      ->  Nested Loop Semi Join  (cost=0.99..96.55 rows=3 width=1384) (actual time=0.059..0.060 rows=1 loops=1)
                                            Buffers: shared hit=9
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.032..0.032 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.025..0.025 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=5
                          ->  Limit  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.221..0.229 rows=1 loops=1)
                                Buffers: shared hit=38
                                ->  Sort  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.221..0.228 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=38
                                      ->  Result  (cost=36.97..45.49 rows=1 width=1392) (actual time=0.140..0.210 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=38
                                            ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=1384) (actual time=0.137..0.205 rows=2 loops=1)
                                                  Buffers: shared hit=38
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.109..0.145 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.077..0.083 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.056..0.069 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.045..0.046 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.022..0.023 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.018..0.018 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.018..0.018 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.677 ms
Execution Time: 6.754 ms
```

### text eq 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=5215.69..5215.70 rows=1 width=8) (actual time=36.037..36.041 rows=1 loops=1)
  Buffers: shared hit=11141
  ->  Limit  (cost=0.99..5203.17 rows=1001 width=4) (actual time=0.077..35.536 rows=1001 loops=1)
        Buffers: shared hit=11141
        ->  Result  (cost=0.99..212204.34 rows=40832 width=4) (actual time=0.075..35.278 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=11141
              ->  Nested Loop Semi Join  (cost=0.99..212204.34 rows=40832 width=0) (actual time=0.065..34.945 rows=1001 loops=1)
                    Buffers: shared hit=11141
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.036..2.158 rows=2394 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1564
                    ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..0.97 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=2394)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r.id))
                          Heap Fetches: 0
                          Buffers: shared hit=9577
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.140 ms
Execution Time: 36.124 ms
```

### text eq 40%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.55..655.95 rows=51 width=1426) (actual time=0.174..2.269 rows=51 loops=1)
  Buffers: shared hit=776
  ->  Result  (cost=1.55..523934.87 rows=40832 width=1426) (actual time=0.170..2.252 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=776
        ->  Nested Loop Left Join  (cost=1.55..523628.63 rows=40832 width=1399) (actual time=0.144..2.172 rows=51 loops=1)
              Buffers: shared hit=776
              ->  Nested Loop  (cost=1.25..510269.10 rows=40832 width=1391) (actual time=0.117..1.816 rows=51 loops=1)
                    Buffers: shared hit=623
                    ->  Nested Loop Semi Join  (cost=0.98..509750.40 rows=40832 width=1384) (actual time=0.089..1.735 rows=51 loops=1)
                          Buffers: shared hit=620
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.050..0.396 rows=123 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                Buffers: shared hit=127
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..0.97 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=123)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r.id))
                                Heap Fetches: 0
                                Buffers: shared hit=493
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.017..0.020 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.199 ms
Execution Time: 2.417 ms
```

### text eq 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2306.85..2306.86 rows=1 width=8) (actual time=33.233..33.247 rows=1 loops=1)
  Buffers: shared hit=18590
  ->  Limit  (cost=669.37..2306.05 rows=64 width=4) (actual time=4.747..33.046 rows=499 loops=1)
        Buffers: shared hit=18590
        ->  Result  (cost=669.37..2306.05 rows=64 width=4) (actual time=4.744..32.928 rows=499 loops=1)
              Buffers: shared hit=18590
              ->  Append  (cost=669.37..2305.41 rows=64 width=0) (actual time=4.743..32.787 rows=499 loops=1)
                    Buffers: shared hit=18590
                    ->  Subquery Scan on "*SELECT* 1"  (cost=669.37..1226.32 rows=61 width=0) (actual time=4.742..24.998 rows=493 loops=1)
                          Buffers: shared hit=15782
                          ->  Limit  (cost=669.37..1225.71 rows=61 width=4) (actual time=4.740..24.810 rows=493 loops=1)
                                Buffers: shared hit=15782
                                ->  Result  (cost=669.37..1225.71 rows=61 width=4) (actual time=4.739..24.698 rows=493 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15782
                                      ->  Hash Semi Join  (cost=669.37..1225.71 rows=61 width=0) (actual time=4.725..24.550 rows=493 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15782
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.043..11.506 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=570.14..570.14 rows=7893 width=32) (actual time=4.612..4.613 rows=7980 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..570.14 rows=7893 width=32) (actual time=0.033..2.517 rows=7980 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..936.71 rows=1 width=0) (actual time=1.127..6.914 rows=6 loops=1)
                          Buffers: shared hit=2460
                          ->  Limit  (cost=0.99..936.70 rows=1 width=4) (actual time=1.126..6.907 rows=6 loops=1)
                                Buffers: shared hit=2460
                                ->  Result  (cost=0.99..936.70 rows=1 width=4) (actual time=1.123..6.900 rows=6 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2460
                                      ->  Nested Loop Semi Join  (cost=0.99..936.70 rows=1 width=0) (actual time=1.101..6.872 rows=6 loops=1)
                                            Buffers: shared hit=2460
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.044..1.726 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=484)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1937
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..96.56 rows=1 width=0) (actual time=0.590..0.591 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..96.55 rows=1 width=4) (actual time=0.588..0.589 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..96.55 rows=1 width=4) (actual time=0.586..0.586 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..96.55 rows=1 width=0) (actual time=0.571..0.572 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.027..0.143 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.50 rows=1 width=0) (actual time=0.118..0.121 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.49 rows=1 width=4) (actual time=0.117..0.120 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.49 rows=1 width=4) (actual time=0.116..0.118 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=0) (actual time=0.110..0.113 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.071..0.093 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.058 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.045 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.339 ms
Execution Time: 33.479 ms
```

### text eq 2%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=15481.63..16473.71 rows=1 width=1434) (actual time=57.900..58.286 rows=51 loops=1)
  Buffers: shared hit=25708
  ->  Nested Loop  (cost=15481.33..16465.38 rows=1 width=1399) (actual time=57.877..58.109 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=25555
        ->  Subquery Scan on x  (cost=15481.05..16457.07 rows=1 width=1392) (actual time=57.859..57.969 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=25402
              ->  Limit  (cost=15481.05..16455.92 rows=51 width=1392) (actual time=57.855..57.918 rows=51 loops=1)
                    Buffers: shared hit=25402
                    ->  Merge Append  (cost=15481.05..16513.27 rows=54 width=1392) (actual time=57.853..57.908 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=25402
                          ->  Limit  (cost=15433.53..15433.66 rows=51 width=1392) (actual time=56.527..56.553 rows=51 loops=1)
                                Buffers: shared hit=24594
                                ->  Sort  (cost=15433.53..15433.68 rows=61 width=1392) (actual time=56.525..56.545 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24594
                                      ->  Result  (cost=669.37..15431.72 rows=61 width=1392) (actual time=5.526..55.574 rows=493 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=24594
                                            ->  Hash Semi Join  (cost=669.37..15431.72 rows=61 width=1384) (actual time=5.515..55.313 rows=493 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=24594
                                                  ->  Index Scan using records_list_group on records r  (cost=0.55..14746.00 rows=6151 width=1384) (actual time=0.033..41.329 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=24492
                                                  ->  Hash  (cost=570.14..570.14 rows=7893 width=32) (actual time=5.348..5.350 rows=7980 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 563kB
                                                        Buffers: shared hit=102
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..570.14 rows=7893 width=32) (actual time=0.029..2.789 rows=7980 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=102
                          ->  Limit  (cost=0.99..936.70 rows=1 width=1392) (actual time=0.745..0.748 rows=1 loops=1)
                                Buffers: shared hit=460
                                ->  Result  (cost=0.99..936.70 rows=1 width=1392) (actual time=0.742..0.744 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=460
                                      ->  Nested Loop Semi Join  (cost=0.99..936.70 rows=1 width=1384) (actual time=0.695..0.697 rows=1 loops=1)
                                            Buffers: shared hit=460
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.069..0.234 rows=90 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 5
                                                  Buffers: shared hit=99
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=90)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=361
                          ->  Limit  (cost=0.99..96.55 rows=1 width=1392) (actual time=0.427..0.429 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..96.55 rows=1 width=1392) (actual time=0.426..0.428 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..96.55 rows=1 width=1384) (actual time=0.420..0.421 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.029..0.131 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.146..0.150 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.146..0.149 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.49 rows=1 width=1392) (actual time=0.120..0.122 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=1384) (actual time=0.116..0.118 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.074..0.094 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.056 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.045 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.010..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.744 ms
Execution Time: 58.542 ms
```

### text eq 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=7976.08..7976.09 rows=1 width=8) (actual time=36.500..36.505 rows=1 loops=1)
  Buffers: shared hit=31633
  ->  Limit  (cost=669.25..7963.57 rows=1001 width=4) (actual time=5.353..36.235 rows=1001 loops=1)
        Buffers: shared hit=31633
        ->  Result  (cost=669.25..15279.75 rows=2005 width=4) (actual time=5.351..36.056 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=31633
              ->  Hash Semi Join  (cost=669.25..15279.75 rows=2005 width=0) (actual time=5.340..35.816 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=31633
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.036..17.736 rows=48655 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=31532
                    ->  Hash  (cost=570.14..570.14 rows=7893 width=32) (actual time=5.262..5.265 rows=7980 loops=1)
                          Buckets: 8192  Batches: 1  Memory Usage: 563kB
                          Buffers: shared hit=101
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..570.14 rows=7893 width=32) (actual time=0.030..2.860 rows=7980 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=101
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.666 ms
Execution Time: 36.579 ms
```

### text eq 2%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1014.54..1014.66 rows=51 width=1426) (actual time=88.992..89.021 rows=51 loops=1)
  Buffers: shared hit=32079
  ->  Sort  (cost=1014.54..1019.55 rows=2005 width=1426) (actual time=88.990..89.013 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 205kB
        Buffers: shared hit=32079
        ->  Result  (cost=755.01..947.65 rows=2005 width=1426) (actual time=5.092..81.285 rows=7980 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32079
              ->  Hash Left Join  (cost=755.01..932.61 rows=2005 width=1399) (actual time=5.074..74.960 rows=7980 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32079
                    ->  Nested Loop  (cost=610.31..782.64 rows=2005 width=1391) (actual time=4.875..70.277 rows=7980 loops=1)
                          Buffers: shared hit=32025
                          ->  Nested Loop  (cost=609.88..775.34 rows=1 width=71) (actual time=4.840..10.414 rows=7980 loops=1)
                                Buffers: shared hit=105
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.025..0.028 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=609.61..688.33 rows=7872 width=32) (actual time=4.813..8.376 rows=7980 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=102
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..570.14 rows=7893 width=32) (actual time=0.029..1.818 rows=7980 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=102
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.82 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=7980)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=31920
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.185..0.188 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.049..0.171 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.036..0.146 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.019..0.019 rows=53 loops=1)
                                            Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.421 ms
Execution Time: 89.201 ms
```

### text eq 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1114.79..1114.80 rows=1 width=8) (actual time=18.145..18.163 rows=1 loops=1)
  Buffers: shared hit=16353
  ->  Limit  (cost=48.96..1114.70 rows=7 width=4) (actual time=0.818..18.138 rows=46 loops=1)
        Buffers: shared hit=16353
        ->  Result  (cost=48.96..1114.70 rows=7 width=4) (actual time=0.816..18.121 rows=46 loops=1)
              Buffers: shared hit=16353
              ->  Append  (cost=48.96..1114.63 rows=7 width=0) (actual time=0.814..18.100 rows=46 loops=1)
                    Buffers: shared hit=16353
                    ->  Subquery Scan on "*SELECT* 1"  (cost=48.96..604.64 rows=4 width=0) (actual time=0.813..15.192 rows=45 loops=1)
                          Buffers: shared hit=15696
                          ->  Limit  (cost=48.96..604.60 rows=4 width=4) (actual time=0.811..15.174 rows=45 loops=1)
                                Buffers: shared hit=15696
                                ->  Result  (cost=48.96..604.60 rows=4 width=4) (actual time=0.810..15.160 rows=45 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15696
                                      ->  Hash Semi Join  (cost=48.96..604.60 rows=4 width=0) (actual time=0.796..15.128 rows=45 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15696
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.055..9.759 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.707..0.708 rows=1017 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                  Buffers: shared hit=15
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.054..0.424 rows=1017 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15
                    ->  Subquery Scan on "*SELECT* 2"  (cost=48.84..384.49 rows=1 width=0) (actual time=1.570..2.032 rows=1 loops=1)
                          Buffers: shared hit=538
                          ->  Limit  (cost=48.84..384.48 rows=1 width=4) (actual time=1.569..2.029 rows=1 loops=1)
                                Buffers: shared hit=538
                                ->  Result  (cost=48.84..384.48 rows=1 width=4) (actual time=1.566..2.025 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=538
                                      ->  Hash Semi Join  (cost=48.84..384.48 rows=1 width=0) (actual time=1.544..2.002 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=538
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.043..1.154 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.711..0.711 rows=1017 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                  Buffers: shared hit=15
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..41.31 rows=567 width=32) (actual time=0.038..0.394 rows=1017 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15
                    ->  Subquery Scan on "*SELECT* 3"  (cost=37.93..79.96 rows=1 width=0) (actual time=0.689..0.692 rows=0 loops=1)
                          Buffers: shared hit=82
                          ->  Limit  (cost=37.93..79.95 rows=1 width=4) (actual time=0.688..0.690 rows=0 loops=1)
                                Buffers: shared hit=82
                                ->  Result  (cost=37.93..79.95 rows=1 width=4) (actual time=0.686..0.688 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=82
                                      ->  Merge Semi Join  (cost=37.93..79.95 rows=1 width=0) (actual time=0.674..0.676 rows=0 loops=1)
                                            Merge Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=82
                                            ->  Sort  (cost=37.14..37.17 rows=13 width=32) (actual time=0.191..0.200 rows=61 loops=1)
                                                  Sort Key: r_2.id
                                                  Sort Method: quicksort  Memory: 28kB
                                                  Buffers: shared hit=67
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.029..0.150 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..41.31 rows=567 width=32) (actual time=0.027..0.305 rows=1016 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.50 rows=1 width=0) (actual time=0.157..0.161 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.49 rows=1 width=4) (actual time=0.156..0.160 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.49 rows=1 width=4) (actual time=0.155..0.158 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=0) (actual time=0.151..0.153 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.094..0.125 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.071..0.074 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.051..0.060 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.037..0.037 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.021..0.021 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.014..0.014 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.789 ms
Execution Time: 18.473 ms
```

### text eq 0.25%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=5264.07..5280.43 rows=1 width=1434) (actual time=14.381..14.908 rows=46 loops=1)
  Buffers: shared hit=5019
  ->  Nested Loop  (cost=5263.77..5272.09 rows=1 width=1399) (actual time=14.354..14.664 rows=46 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4881
        ->  Subquery Scan on x  (cost=5263.49..5263.78 rows=1 width=1392) (actual time=14.337..14.475 rows=46 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4743
              ->  Limit  (cost=5263.49..5263.62 rows=7 width=1392) (actual time=14.332..14.410 rows=46 loops=1)
                    Buffers: shared hit=4743
                    ->  Merge Append  (cost=5263.49..5263.62 rows=7 width=1392) (actual time=14.330..14.397 rows=46 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4743
                          ->  Limit  (cost=4748.11..4748.12 rows=4 width=1392) (actual time=10.865..10.897 rows=45 loops=1)
                                Buffers: shared hit=4084
                                ->  Sort  (cost=4748.11..4748.12 rows=4 width=1392) (actual time=10.863..10.886 rows=45 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 115kB
                                      Buffers: shared hit=4084
                                      ->  Result  (cost=44.58..4748.07 rows=4 width=1392) (actual time=1.253..10.755 rows=45 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=4084
                                            ->  Nested Loop  (cost=44.58..4748.07 rows=4 width=1384) (actual time=1.243..10.713 rows=45 loops=1)
                                                  Buffers: shared hit=4084
                                                  ->  HashAggregate  (cost=44.15..49.82 rows=567 width=32) (actual time=1.066..1.448 rows=1017 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=16
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.058..0.371 rows=1017 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=16
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.28 rows=1 width=1384) (actual time=0.009..0.009 rows=0 loops=1017)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=4068
                          ->  Limit  (cost=384.49..384.50 rows=1 width=1392) (actual time=2.261..2.266 rows=1 loops=1)
                                Buffers: shared hit=539
                                ->  Sort  (cost=384.49..384.50 rows=1 width=1392) (actual time=2.257..2.261 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=539
                                      ->  Result  (cost=48.84..384.48 rows=1 width=1392) (actual time=1.672..2.236 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=539
                                            ->  Hash Semi Join  (cost=48.84..384.48 rows=1 width=1384) (actual time=1.652..2.212 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=539
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.029..1.259 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.800..0.801 rows=1017 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                        Buffers: shared hit=16
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_1  (cost=0.55..41.31 rows=567 width=32) (actual time=0.039..0.404 rows=1017 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=16
                          ->  Limit  (cost=85.35..85.36 rows=1 width=1392) (actual time=0.986..0.989 rows=0 loops=1)
                                Buffers: shared hit=83
                                ->  Sort  (cost=85.35..85.36 rows=1 width=1392) (actual time=0.984..0.987 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=83
                                      ->  Result  (cost=48.84..85.34 rows=1 width=1392) (actual time=0.973..0.975 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=83
                                            ->  Hash Semi Join  (cost=48.84..85.34 rows=1 width=1384) (actual time=0.961..0.963 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=83
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.028..0.175 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.761..0.762 rows=1017 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                        Buffers: shared hit=16
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_2  (cost=0.55..41.31 rows=567 width=32) (actual time=0.033..0.425 rows=1017 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=16
                          ->  Limit  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.208..0.212 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.207..0.211 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.49 rows=1 width=1392) (actual time=0.186..0.190 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=1384) (actual time=0.180..0.183 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.117..0.144 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.084..0.088 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.063..0.073 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.049..0.050 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.024..0.024 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.016..0.016 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=46)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=138
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=46)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=138
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 5.821 ms
Execution Time: 15.339 ms
```

### text eq 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=4747.03..4747.04 rows=1 width=8) (actual time=11.223..11.227 rows=1 loops=1)
  Buffers: shared hit=4019
  ->  Limit  (cost=44.58..4745.23 rows=144 width=4) (actual time=1.169..10.997 rows=1001 loops=1)
        Buffers: shared hit=4019
        ->  Result  (cost=44.58..4745.23 rows=144 width=4) (actual time=1.167..10.758 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=4019
              ->  Nested Loop  (cost=44.58..4745.23 rows=144 width=0) (actual time=1.153..10.476 rows=1001 loops=1)
                    Buffers: shared hit=4019
                    ->  HashAggregate  (cost=44.15..49.82 rows=567 width=32) (actual time=1.120..1.492 rows=1001 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=15
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.195..0.512 rows=1017 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=15
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=1001)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=4004
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.907 ms
Execution Time: 11.368 ms
```

### text eq 0.25%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=125.10..125.23 rows=51 width=1426) (actual time=17.426..17.449 rows=51 loops=1)
  Buffers: shared hit=7138
  ->  Sort  (cost=125.10..125.46 rows=144 width=1426) (actual time=17.424..17.440 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 175kB
        Buffers: shared hit=7138
        ->  Result  (cost=45.14..120.30 rows=144 width=1426) (actual time=0.769..16.249 rows=1017 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=7138
              ->  Nested Loop Left Join  (cost=45.14..119.22 rows=144 width=1399) (actual time=0.748..15.157 rows=1017 loops=1)
                    Buffers: shared hit=7138
                    ->  Nested Loop  (cost=44.84..72.09 rows=144 width=1391) (actual time=0.725..10.257 rows=1017 loops=1)
                          Buffers: shared hit=4087
                          ->  Nested Loop  (cost=44.42..63.78 rows=1 width=71) (actual time=0.699..1.426 rows=1017 loops=1)
                                Buffers: shared hit=19
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.036..0.039 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=44.15..49.82 rows=567 width=32) (actual time=0.661..1.113 rows=1017 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=16
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.038..0.263 rows=1017 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=16
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=1017)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=4068
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=1017)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=3051
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=1017)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=3051
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.260 ms
Execution Time: 17.595 ms
```

### text eq 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1114.79..1114.80 rows=1 width=8) (actual time=14.913..14.933 rows=1 loops=1)
  Buffers: shared hit=16323
  ->  Limit  (cost=48.96..1114.70 rows=7 width=4) (actual time=9.196..14.922 rows=2 loops=1)
        Buffers: shared hit=16323
        ->  Result  (cost=48.96..1114.70 rows=7 width=4) (actual time=9.194..14.918 rows=2 loops=1)
              Buffers: shared hit=16323
              ->  Append  (cost=48.96..1114.63 rows=7 width=0) (actual time=9.193..14.914 rows=2 loops=1)
                    Buffers: shared hit=16323
                    ->  Subquery Scan on "*SELECT* 1"  (cost=48.96..604.64 rows=4 width=0) (actual time=9.191..13.261 rows=2 loops=1)
                          Buffers: shared hit=15686
                          ->  Limit  (cost=48.96..604.60 rows=4 width=4) (actual time=9.189..13.256 rows=2 loops=1)
                                Buffers: shared hit=15686
                                ->  Result  (cost=48.96..604.60 rows=4 width=4) (actual time=9.187..13.252 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15686
                                      ->  Hash Semi Join  (cost=48.96..604.60 rows=4 width=0) (actual time=9.174..13.236 rows=2 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15686
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.047..9.777 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.067..0.068 rows=48 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.034..0.048 rows=48 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 2"  (cost=48.84..384.49 rows=1 width=0) (actual time=1.152..1.155 rows=0 loops=1)
                          Buffers: shared hit=528
                          ->  Limit  (cost=48.84..384.48 rows=1 width=4) (actual time=1.150..1.153 rows=0 loops=1)
                                Buffers: shared hit=528
                                ->  Result  (cost=48.84..384.48 rows=1 width=4) (actual time=1.148..1.150 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=528
                                      ->  Hash Semi Join  (cost=48.84..384.48 rows=1 width=0) (actual time=1.130..1.132 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=528
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.037..0.997 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.051..0.052 rows=48 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..41.31 rows=567 width=32) (actual time=0.029..0.038 rows=48 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 3"  (cost=37.93..79.96 rows=1 width=0) (actual time=0.313..0.315 rows=0 loops=1)
                          Buffers: shared hit=72
                          ->  Limit  (cost=37.93..79.95 rows=1 width=4) (actual time=0.312..0.314 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Result  (cost=37.93..79.95 rows=1 width=4) (actual time=0.311..0.312 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=72
                                      ->  Merge Semi Join  (cost=37.93..79.95 rows=1 width=0) (actual time=0.304..0.305 rows=0 loops=1)
                                            Merge Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=72
                                            ->  Sort  (cost=37.14..37.17 rows=13 width=32) (actual time=0.196..0.204 rows=61 loops=1)
                                                  Sort Key: r_2.id
                                                  Sort Method: quicksort  Memory: 28kB
                                                  Buffers: shared hit=67
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.024..0.154 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..41.31 rows=567 width=32) (actual time=0.018..0.073 rows=48 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.50 rows=1 width=0) (actual time=0.167..0.171 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.49 rows=1 width=4) (actual time=0.166..0.170 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.49 rows=1 width=4) (actual time=0.165..0.168 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=0) (actual time=0.157..0.160 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.103..0.131 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.078..0.082 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.058..0.067 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.042..0.043 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.557 ms
Execution Time: 15.175 ms
```

### text eq 0.01%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=5264.07..5280.43 rows=1 width=1434) (actual time=2.935..2.975 rows=2 loops=1)
  Buffers: shared hit=846
  ->  Nested Loop  (cost=5263.77..5272.09 rows=1 width=1399) (actual time=2.904..2.937 rows=2 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=840
        ->  Subquery Scan on x  (cost=5263.49..5263.78 rows=1 width=1392) (actual time=2.858..2.882 rows=2 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=834
              ->  Limit  (cost=5263.49..5263.62 rows=7 width=1392) (actual time=2.852..2.873 rows=2 loops=1)
                    Buffers: shared hit=834
                    ->  Merge Append  (cost=5263.49..5263.62 rows=7 width=1392) (actual time=2.849..2.869 rows=2 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=834
                          ->  Limit  (cost=4748.11..4748.12 rows=4 width=1392) (actual time=0.786..0.793 rows=2 loops=1)
                                Buffers: shared hit=197
                                ->  Sort  (cost=4748.11..4748.12 rows=4 width=1392) (actual time=0.785..0.791 rows=2 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=197
                                      ->  Result  (cost=44.58..4748.07 rows=4 width=1392) (actual time=0.327..0.772 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=197
                                            ->  Nested Loop  (cost=44.58..4748.07 rows=4 width=1384) (actual time=0.314..0.757 rows=2 loops=1)
                                                  Buffers: shared hit=197
                                                  ->  HashAggregate  (cost=44.15..49.82 rows=567 width=32) (actual time=0.121..0.149 rows=48 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 49kB
                                                        Buffers: shared hit=5
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.062..0.079 rows=48 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.28 rows=1 width=1384) (actual time=0.012..0.012 rows=0 loops=48)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=192
                          ->  Limit  (cost=384.49..384.50 rows=1 width=1392) (actual time=1.577..1.581 rows=0 loops=1)
                                Buffers: shared hit=528
                                ->  Sort  (cost=384.49..384.50 rows=1 width=1392) (actual time=1.576..1.579 rows=0 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=528
                                      ->  Result  (cost=48.84..384.48 rows=1 width=1392) (actual time=1.567..1.570 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=528
                                            ->  Hash Semi Join  (cost=48.84..384.48 rows=1 width=1384) (actual time=1.558..1.560 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=528
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.025..1.375 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.050..0.051 rows=48 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                        Buffers: shared hit=5
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_1  (cost=0.55..41.31 rows=567 width=32) (actual time=0.021..0.033 rows=48 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                          ->  Limit  (cost=85.35..85.36 rows=1 width=1392) (actual time=0.286..0.288 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Sort  (cost=85.35..85.36 rows=1 width=1392) (actual time=0.284..0.286 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=72
                                      ->  Result  (cost=48.84..85.34 rows=1 width=1392) (actual time=0.275..0.276 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=72
                                            ->  Hash Semi Join  (cost=48.84..85.34 rows=1 width=1384) (actual time=0.266..0.267 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=72
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.029..0.190 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=41.31..41.31 rows=567 width=32) (actual time=0.055..0.056 rows=48 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                        Buffers: shared hit=5
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_2  (cost=0.55..41.31 rows=567 width=32) (actual time=0.024..0.037 rows=48 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                          ->  Limit  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.196..0.200 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.50..45.51 rows=1 width=1392) (actual time=0.195..0.199 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.49 rows=1 width=1392) (actual time=0.191..0.195 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.49 rows=1 width=1384) (actual time=0.187..0.190 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.122..0.150 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.066..0.069 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.059 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.036..0.038 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.014 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.024..0.024 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.02 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.021..0.021 rows=1 loops=2)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=6
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.009..0.009 rows=1 loops=2)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=6
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.500 ms
Execution Time: 3.417 ms
```

### text eq 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=4747.03..4747.04 rows=1 width=8) (actual time=0.541..0.542 rows=1 loops=1)
  Buffers: shared hit=197
  ->  Limit  (cost=44.58..4745.23 rows=144 width=4) (actual time=0.134..0.531 rows=48 loops=1)
        Buffers: shared hit=197
        ->  Result  (cost=44.58..4745.23 rows=144 width=4) (actual time=0.132..0.522 rows=48 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=197
              ->  Nested Loop  (cost=44.58..4745.23 rows=144 width=0) (actual time=0.121..0.502 rows=48 loops=1)
                    Buffers: shared hit=197
                    ->  HashAggregate  (cost=44.15..49.82 rows=567 width=32) (actual time=0.093..0.108 rows=48 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 49kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.052..0.062 rows=48 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=48)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=192
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.966 ms
Execution Time: 0.623 ms
```

### text eq 0.01%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=125.10..125.23 rows=51 width=1426) (actual time=0.660..0.669 rows=48 loops=1)
  Buffers: shared hit=344
  ->  Sort  (cost=125.10..125.46 rows=144 width=1426) (actual time=0.659..0.663 rows=48 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 121kB
        Buffers: shared hit=344
        ->  Result  (cost=45.14..120.30 rows=144 width=1426) (actual time=0.102..0.612 rows=48 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=344
              ->  Nested Loop Left Join  (cost=45.14..119.22 rows=144 width=1399) (actual time=0.090..0.577 rows=48 loops=1)
                    Buffers: shared hit=344
                    ->  Nested Loop  (cost=44.84..72.09 rows=144 width=1391) (actual time=0.078..0.421 rows=48 loops=1)
                          Buffers: shared hit=200
                          ->  Nested Loop  (cost=44.42..63.78 rows=1 width=71) (actual time=0.062..0.083 rows=48 loops=1)
                                Buffers: shared hit=8
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.020..0.021 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=44.15..49.82 rows=567 width=32) (actual time=0.041..0.053 rows=48 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 49kB
                                      Buffers: shared hit=5
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..41.31 rows=567 width=32) (actual time=0.022..0.027 rows=48 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'ecda677d-4f54-4bb3-89b0-42b6757b1da6'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=5
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=48)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=192
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=48)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=144
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=48)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=144
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.781 ms
Execution Time: 0.755 ms
```

### text in (2 values) 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=16757.79..16757.80 rows=1 width=8) (actual time=113.928..113.940 rows=1 loops=1)
  Buffers: shared hit=5049, temp written=526
  I/O Timings: temp write=6.936
  ->  Limit  (cost=13726.40..16745.28 rows=1001 width=4) (actual time=109.411..113.820 rows=1001 loops=1)
        Buffers: shared hit=5049, temp written=526
        I/O Timings: temp write=6.936
        ->  Result  (cost=13726.40..16832.74 rows=1030 width=4) (actual time=109.408..113.686 rows=1001 loops=1)
              Buffers: shared hit=5049, temp written=526
              I/O Timings: temp write=6.936
              ->  Append  (cost=13726.40..16822.44 rows=1030 width=0) (actual time=109.407..113.515 rows=1001 loops=1)
                    Buffers: shared hit=5049, temp written=526
                    I/O Timings: temp write=6.936
                    ->  Subquery Scan on "*SELECT* 1"  (cost=13726.40..15180.01 rows=1001 width=0) (actual time=109.405..113.373 rows=1001 loops=1)
                          Buffers: shared hit=5049, temp written=526
                          I/O Timings: temp write=6.936
                          ->  Limit  (cost=13726.40..15170.00 rows=1001 width=4) (actual time=109.404..113.231 rows=1001 loops=1)
                                Buffers: shared hit=5049, temp written=526
                                I/O Timings: temp write=6.936
                                ->  Result  (cost=13726.40..15439.69 rows=1188 width=4) (actual time=109.402..113.125 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=5049, temp written=526
                                      I/O Timings: temp write=6.936
                                      ->  Hash Semi Join  (cost=13726.40..15439.69 rows=1188 width=0) (actual time=109.386..112.962 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=5049, temp written=526
                                            I/O Timings: temp write=6.936
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.060..1.674 rows=4791 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=3117
                                            ->  Hash  (cost=10737.50..10737.50 rows=154507 width=32) (actual time=108.252..108.253 rows=160142 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 6000kB
                                                  Buffers: shared hit=1932, temp written=511
                                                  I/O Timings: temp write=6.807
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..10737.50 rows=154507 width=32) (actual time=0.068..43.765 rows=160142 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p40a,p40b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1932
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1444.58 rows=25 width=0) (never executed)
                          ->  Limit  (cost=0.86..1444.33 rows=25 width=4) (never executed)
                                ->  Result  (cost=0.86..1444.33 rows=25 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1444.33 rows=25 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_1  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..146.95 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..146.92 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..146.92 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..146.92 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.74 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.73 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.73 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.733 ms
Execution Time: 115.956 ms
```

### text in (2 values) 40%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=48.96..10907.87 rows=1 width=1434) (actual time=0.676..5.234 rows=51 loops=1)
  Buffers: shared hit=2910
  ->  Nested Loop  (cost=48.66..10899.53 rows=1 width=1399) (actual time=0.654..5.010 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2757
        ->  Subquery Scan on x  (cost=48.38..10891.22 rows=1 width=1392) (actual time=0.631..4.828 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2604
              ->  Limit  (cost=48.38..10890.07 rows=51 width=1392) (actual time=0.627..4.762 rows=51 loops=1)
                    Buffers: shared hit=2604
                    ->  Merge Append  (cost=48.38..17054.96 rows=80 width=1392) (actual time=0.625..4.749 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2604
                          ->  Limit  (cost=0.86..15416.71 rows=51 width=1392) (actual time=0.256..4.308 rows=50 loops=1)
                                Buffers: shared hit=2500
                                ->  Result  (cost=0.86..359099.49 rows=1188 width=1392) (actual time=0.255..4.297 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2500
                                      ->  Nested Loop Semi Join  (cost=0.86..359099.49 rows=1188 width=1384) (actual time=0.245..4.265 rows=50 loops=1)
                                            Buffers: shared hit=2500
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..314238.64 rows=6151 width=1384) (actual time=0.145..3.348 rows=130 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1826
                                                  Buffers: shared hit=1980
                                            ->  Index Scan using rfv_text_pkey on rfv_text v  (cost=0.43..7.28 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=130)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=520
                          ->  Limit  (cost=0.86..1444.33 rows=25 width=1392) (actual time=0.049..0.091 rows=2 loops=1)
                                Buffers: shared hit=28
                                ->  Result  (cost=0.86..1444.33 rows=25 width=1392) (actual time=0.048..0.089 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=28
                                      ->  Nested Loop Semi Join  (cost=0.86..1444.33 rows=25 width=1384) (actual time=0.043..0.082 rows=2 loops=1)
                                            Buffers: shared hit=28
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.024..0.033 rows=5 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=8
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_1  (cost=0.43..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=5)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=20
                          ->  Limit  (cost=0.86..146.92 rows=3 width=1392) (actual time=0.136..0.140 rows=1 loops=1)
                                Buffers: shared hit=39
                                ->  Result  (cost=0.86..146.92 rows=3 width=1392) (actual time=0.136..0.139 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=39
                                      ->  Nested Loop Semi Join  (cost=0.86..146.92 rows=3 width=1384) (actual time=0.133..0.134 rows=1 loops=1)
                                            Buffers: shared hit=39
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.033..0.055 rows=7 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=11
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=7)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=28
                          ->  Limit  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.180..0.189 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.180..0.187 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.73 rows=1 width=1392) (actual time=0.118..0.166 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=1384) (actual time=0.114..0.160 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.095..0.121 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.070..0.074 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.053..0.062 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.041..0.043 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.023..0.023 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.018 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.014..0.014 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                        Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.123 ms
Execution Time: 5.709 ms
```

### text in (2 values) 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=6456.23..6456.24 rows=1 width=8) (actual time=21.417..21.421 rows=1 loops=1)
  Buffers: shared hit=11329
  ->  Limit  (cost=0.86..6443.71 rows=1001 width=4) (actual time=0.103..21.079 rows=1001 loops=1)
        Buffers: shared hit=11329
        ->  Result  (cost=0.86..252610.71 rows=39247 width=4) (actual time=0.101..20.841 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=11329
              ->  Nested Loop Semi Join  (cost=0.86..252610.71 rows=39247 width=0) (actual time=0.089..20.512 rows=1001 loops=1)
                    Buffers: shared hit=11329
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.042..1.917 rows=2436 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1585
                    ->  Index Scan using rfv_text_pkey on rfv_text v  (cost=0.43..1.16 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=2436)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                          Filter: (value = ANY ('{p40a,p40b}'::text[]))
                          Rows Removed by Filter: 1
                          Buffers: shared hit=9744
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.062 ms
Execution Time: 21.496 ms
```

### text in (2 values) 40%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.43..734.03 rows=51 width=1426) (actual time=0.170..1.342 rows=51 loops=1)
  Buffers: shared hit=680
  ->  Result  (cost=1.43..563775.10 rows=39247 width=1426) (actual time=0.168..1.330 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=680
        ->  Nested Loop Left Join  (cost=1.43..563480.75 rows=39247 width=1399) (actual time=0.149..1.274 rows=51 loops=1)
              Buffers: shared hit=680
              ->  Nested Loop  (cost=1.12..550639.81 rows=39247 width=1391) (actual time=0.091..0.990 rows=51 loops=1)
                    Buffers: shared hit=527
                    ->  Nested Loop Semi Join  (cost=0.85..550140.93 rows=39247 width=1384) (actual time=0.073..0.932 rows=51 loops=1)
                          Buffers: shared hit=524
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.033..0.256 rows=104 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                Buffers: shared hit=108
                          ->  Index Scan using rfv_text_pkey on rfv_text v  (cost=0.43..1.16 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=104)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                Rows Removed by Filter: 1
                                Buffers: shared hit=416
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.012..0.014 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.094 ms
Execution Time: 1.425 ms
```

### text in (2 values) 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2290.50..2290.51 rows=1 width=8) (actual time=28.157..28.177 rows=1 loops=1)
  Buffers: shared hit=16754
  ->  Limit  (cost=601.94..2289.78 rows=58 width=4) (actual time=4.746..27.982 rows=491 loops=1)
        Buffers: shared hit=16754
        ->  Result  (cost=601.94..2289.78 rows=58 width=4) (actual time=4.744..27.889 rows=491 loops=1)
              Buffers: shared hit=16754
              ->  Append  (cost=601.94..2289.20 rows=58 width=0) (actual time=4.743..27.766 rows=491 loops=1)
                    Buffers: shared hit=16754
                    ->  Subquery Scan on "*SELECT* 1"  (cost=601.94..1158.75 rows=55 width=0) (actual time=4.741..19.492 rows=479 loops=1)
                          Buffers: shared hit=15782
                          ->  Limit  (cost=601.94..1158.20 rows=55 width=4) (actual time=4.740..19.390 rows=479 loops=1)
                                Buffers: shared hit=15782
                                ->  Result  (cost=601.94..1158.20 rows=55 width=4) (actual time=4.739..19.306 rows=479 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15782
                                      ->  Hash Semi Join  (cost=601.94..1158.20 rows=55 width=0) (actual time=4.728..19.192 rows=479 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15782
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.044..8.870 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=512.71..512.71 rows=7093 width=32) (actual time=4.543..4.544 rows=7997 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 564kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..512.71 rows=7093 width=32) (actual time=0.038..2.425 rows=7997 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 2"  (cost=601.81..937.48 rows=1 width=0) (actual time=5.679..7.058 rows=9 loops=1)
                          Buffers: shared hit=624
                          ->  Limit  (cost=601.81..937.47 rows=1 width=4) (actual time=5.677..7.050 rows=9 loops=1)
                                Buffers: shared hit=624
                                ->  Result  (cost=601.81..937.47 rows=1 width=4) (actual time=5.674..7.043 rows=9 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=624
                                      ->  Hash Semi Join  (cost=601.81..937.47 rows=1 width=0) (actual time=5.651..7.016 rows=9 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=624
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.073..1.421 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=512.71..512.71 rows=7093 width=32) (actual time=5.278..5.279 rows=7997 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 564kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..512.71 rows=7093 width=32) (actual time=0.056..2.458 rows=7997 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..146.93 rows=1 width=0) (actual time=0.345..0.922 rows=3 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..146.92 rows=1 width=4) (actual time=0.344..0.918 rows=3 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..146.92 rows=1 width=4) (actual time=0.342..0.914 rows=3 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..146.92 rows=1 width=0) (actual time=0.326..0.895 rows=3 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.114..0.290 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.74 rows=1 width=0) (actual time=0.181..0.185 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.73 rows=1 width=4) (actual time=0.180..0.184 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.73 rows=1 width=4) (actual time=0.178..0.181 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=0) (actual time=0.166..0.169 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.108..0.130 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.077..0.081 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.060..0.068 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.049..0.050 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.030..0.031 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.017..0.018 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.015..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.921 ms
Execution Time: 28.544 ms
```

### text in (2 values) 2%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=16350.51..16506.56 rows=1 width=1434) (actual time=76.246..76.730 rows=51 loops=1)
  Buffers: shared hit=25634
  ->  Nested Loop  (cost=16350.21..16498.23 rows=1 width=1399) (actual time=76.215..76.521 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=25481
        ->  Subquery Scan on x  (cost=16349.93..16489.92 rows=1 width=1392) (actual time=76.193..76.355 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=25328
              ->  Limit  (cost=16349.93..16488.77 rows=51 width=1392) (actual time=76.189..76.302 rows=51 loops=1)
                    Buffers: shared hit=25328
                    ->  Merge Append  (cost=16349.93..16496.94 rows=54 width=1392) (actual time=76.186..76.291 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=25328
                          ->  Limit  (cost=15365.80..15365.93 rows=51 width=1392) (actual time=68.066..68.136 rows=50 loops=1)
                                Buffers: shared hit=24593
                                ->  Sort  (cost=15365.80..15365.94 rows=55 width=1392) (actual time=68.064..68.127 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24593
                                      ->  Result  (cost=601.94..15364.21 rows=55 width=1392) (actual time=5.586..66.960 rows=479 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=24593
                                            ->  Hash Semi Join  (cost=601.94..15364.21 rows=55 width=1384) (actual time=5.575..66.554 rows=479 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=24593
                                                  ->  Index Scan using records_list_group on records r  (cost=0.55..14746.00 rows=6151 width=1384) (actual time=0.038..49.753 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=24492
                                                  ->  Hash  (cost=512.71..512.71 rows=7093 width=32) (actual time=5.147..5.148 rows=7997 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 564kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..512.71 rows=7093 width=32) (actual time=0.031..2.768 rows=7997 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=937.48..937.49 rows=1 width=1392) (actual time=7.687..7.691 rows=2 loops=1)
                                Buffers: shared hit=624
                                ->  Sort  (cost=937.48..937.49 rows=1 width=1392) (actual time=7.683..7.686 rows=2 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 43kB
                                      Buffers: shared hit=624
                                      ->  Result  (cost=601.81..937.47 rows=1 width=1392) (actual time=6.485..7.602 rows=9 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=624
                                            ->  Hash Semi Join  (cost=601.81..937.47 rows=1 width=1384) (actual time=6.470..7.582 rows=9 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=624
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.034..1.164 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=512.71..512.71 rows=7093 width=32) (actual time=5.990..5.991 rows=7997 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 564kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..512.71 rows=7093 width=32) (actual time=0.091..2.835 rows=7997 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=0.86..146.92 rows=1 width=1392) (actual time=0.259..0.261 rows=1 loops=1)
                                Buffers: shared hit=74
                                ->  Result  (cost=0.86..146.92 rows=1 width=1392) (actual time=0.256..0.258 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=74
                                      ->  Nested Loop Semi Join  (cost=0.86..146.92 rows=1 width=1384) (actual time=0.240..0.241 rows=1 loops=1)
                                            Buffers: shared hit=74
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.042..0.073 rows=14 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=18
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=14)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=56
                          ->  Limit  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.165..0.169 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.164..0.168 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.73 rows=1 width=1392) (actual time=0.150..0.153 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=1384) (actual time=0.145..0.148 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.089..0.111 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.067..0.070 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.050..0.058 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.037..0.038 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.011..0.011 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                        Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.213 ms
Execution Time: 77.194 ms
```

### text in (2 values) 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=8728.97..8728.98 rows=1 width=8) (actual time=34.841..34.846 rows=1 loops=1)
  Buffers: shared hit=32718
  ->  Limit  (cost=601.81..8716.46 rows=1001 width=4) (actual time=4.377..34.681 rows=1001 loops=1)
        Buffers: shared hit=32718
        ->  Result  (cost=601.81..15209.80 rows=1802 width=4) (actual time=4.375..34.524 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32718
              ->  Hash Semi Join  (cost=601.81..15209.80 rows=1802 width=0) (actual time=4.366..34.318 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=32718
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.033..16.558 rows=50351 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=32617
                    ->  Hash  (cost=512.71..512.71 rows=7093 width=32) (actual time=4.262..4.264 rows=7997 loops=1)
                          Buckets: 8192  Batches: 1  Memory Usage: 564kB
                          Buffers: shared hit=101
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..512.71 rows=7093 width=32) (actual time=0.035..2.415 rows=7997 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                Heap Fetches: 0
                                Buffers: shared hit=101
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.623 ms
Execution Time: 34.914 ms
```

### text in (2 values) 2%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=928.42..928.55 rows=51 width=1426) (actual time=83.087..83.114 rows=51 loops=1)
  Buffers: shared hit=32146
  ->  Sort  (cost=928.42..932.93 rows=1802 width=1426) (actual time=83.085..83.105 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 189kB
        Buffers: shared hit=32146
        ->  Result  (cost=693.58..868.30 rows=1802 width=1426) (actual time=5.234..75.848 rows=7997 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32146
              ->  Hash Left Join  (cost=693.58..854.79 rows=1802 width=1399) (actual time=5.213..69.906 rows=7997 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32146
                    ->  Nested Loop  (cost=548.87..705.36 rows=1802 width=1391) (actual time=5.040..65.522 rows=7997 loops=1)
                          Buffers: shared hit=32092
                          ->  Nested Loop  (cost=548.45..697.99 rows=1 width=71) (actual time=4.998..9.893 rows=7997 loops=1)
                                Buffers: shared hit=104
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.031..0.036 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=548.18..618.94 rows=7076 width=32) (actual time=4.963..8.073 rows=7997 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=101
                                      ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..512.71 rows=7093 width=32) (actual time=0.066..1.847 rows=7997 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=101
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.93 rows=1 width=1384) (actual time=0.006..0.006 rows=1 loops=7997)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=31988
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.159..0.162 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.041..0.147 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.028..0.123 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.014..0.015 rows=53 loops=1)
                                            Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.654 ms
Execution Time: 83.305 ms
```

### text in (2 values) 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1265.23..1265.24 rows=1 width=8) (actual time=16.656..16.670 rows=1 loops=1)
  Buffers: shared hit=16371
  ->  Limit  (cost=97.15..1265.08 rows=12 width=4) (actual time=0.878..16.637 rows=72 loops=1)
        Buffers: shared hit=16371
        ->  Result  (cost=97.15..1265.08 rows=12 width=4) (actual time=0.877..16.616 rows=72 loops=1)
              Buffers: shared hit=16371
              ->  Append  (cost=97.15..1264.96 rows=12 width=0) (actual time=0.876..16.589 rows=72 loops=1)
                    Buffers: shared hit=16371
                    ->  Subquery Scan on "*SELECT* 1"  (cost=97.15..652.94 rows=9 width=0) (actual time=0.875..14.575 rows=71 loops=1)
                          Buffers: shared hit=15702
                          ->  Limit  (cost=97.15..652.85 rows=9 width=4) (actual time=0.874..14.548 rows=71 loops=1)
                                Buffers: shared hit=15702
                                ->  Result  (cost=97.15..652.85 rows=9 width=4) (actual time=0.873..14.527 rows=71 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15702
                                      ->  Hash Semi Join  (cost=97.15..652.85 rows=9 width=0) (actual time=0.861..14.487 rows=71 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15702
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.042..9.731 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.644..0.645 rows=962 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                  Buffers: shared hit=21
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.030..0.368 rows=962 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=21
                    ->  Subquery Scan on "*SELECT* 2"  (cost=97.02..432.68 rows=1 width=0) (actual time=1.214..1.321 rows=1 loops=1)
                          Buffers: shared hit=544
                          ->  Limit  (cost=97.02..432.67 rows=1 width=4) (actual time=1.213..1.319 rows=1 loops=1)
                                Buffers: shared hit=544
                                ->  Result  (cost=97.02..432.67 rows=1 width=4) (actual time=1.211..1.317 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=544
                                      ->  Hash Semi Join  (cost=97.02..432.67 rows=1 width=0) (actual time=1.198..1.302 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=544
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.029..0.805 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.423..0.423 rows=962 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                  Buffers: shared hit=21
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.025..0.243 rows=962 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=21
                    ->  Subquery Scan on "*SELECT* 3"  (cost=97.02..133.54 rows=1 width=0) (actual time=0.541..0.542 rows=0 loops=1)
                          Buffers: shared hit=88
                          ->  Limit  (cost=97.02..133.53 rows=1 width=4) (actual time=0.540..0.541 rows=0 loops=1)
                                Buffers: shared hit=88
                                ->  Result  (cost=97.02..133.53 rows=1 width=4) (actual time=0.539..0.540 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=88
                                      ->  Hash Semi Join  (cost=97.02..133.53 rows=1 width=0) (actual time=0.530..0.531 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=88
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.018..0.106 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.409..0.410 rows=962 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                  Buffers: shared hit=21
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.015..0.229 rows=962 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=21
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.74 rows=1 width=0) (actual time=0.121..0.124 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.73 rows=1 width=4) (actual time=0.119..0.122 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.73 rows=1 width=4) (actual time=0.118..0.121 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=0) (actual time=0.116..0.117 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.064..0.083 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.050 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.040 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p025a,p025b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.308 ms
Execution Time: 16.878 ms
```

### text in (2 values) 0.25%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=9867.22..9883.77 rows=1 width=1434) (actual time=11.081..11.509 rows=51 loops=1)
  Buffers: shared hit=4844
  ->  Nested Loop  (cost=9866.92..9875.44 rows=1 width=1399) (actual time=11.053..11.304 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4691
        ->  Subquery Scan on x  (cost=9866.64..9867.12 rows=1 width=1392) (actual time=11.036..11.151 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4538
              ->  Limit  (cost=9866.64..9866.85 rows=12 width=1392) (actual time=11.033..11.101 rows=51 loops=1)
                    Buffers: shared hit=4538
                    ->  Merge Append  (cost=9866.64..9866.85 rows=12 width=1392) (actual time=11.031..11.092 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4538
                          ->  Limit  (cost=9254.63..9254.65 rows=9 width=1392) (actual time=8.384..8.410 rows=51 loops=1)
                                Buffers: shared hit=3869
                                ->  Sort  (cost=9254.63..9254.65 rows=9 width=1392) (actual time=8.382..8.401 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 167kB
                                      Buffers: shared hit=3869
                                      ->  Result  (cost=88.56..9254.49 rows=9 width=1392) (actual time=1.047..8.269 rows=71 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=3869
                                            ->  Nested Loop  (cost=88.56..9254.49 rows=9 width=1384) (actual time=1.037..8.224 rows=71 loops=1)
                                                  Buffers: shared hit=3869
                                                  ->  HashAggregate  (cost=88.12..99.41 rows=1128 width=32) (actual time=0.787..1.110 rows=962 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=21
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.077..0.383 rows=962 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=21
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.12 rows=1 width=1384) (actual time=0.007..0.007 rows=0 loops=962)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=3848
                          ->  Limit  (cost=432.68..432.69 rows=1 width=1392) (actual time=1.788..1.791 rows=1 loops=1)
                                Buffers: shared hit=544
                                ->  Sort  (cost=432.68..432.69 rows=1 width=1392) (actual time=1.785..1.788 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=544
                                      ->  Result  (cost=97.02..432.67 rows=1 width=1392) (actual time=1.691..1.770 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=544
                                            ->  Hash Semi Join  (cost=97.02..432.67 rows=1 width=1384) (actual time=1.674..1.753 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=544
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.023..0.887 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.773..0.774 rows=962 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                        Buffers: shared hit=21
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.027..0.326 rows=962 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=21
                          ->  Limit  (cost=133.54..133.54 rows=1 width=1392) (actual time=0.690..0.692 rows=0 loops=1)
                                Buffers: shared hit=88
                                ->  Sort  (cost=133.54..133.54 rows=1 width=1392) (actual time=0.688..0.690 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=88
                                      ->  Result  (cost=97.02..133.53 rows=1 width=1392) (actual time=0.671..0.673 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=88
                                            ->  Hash Semi Join  (cost=97.02..133.53 rows=1 width=1384) (actual time=0.660..0.661 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=88
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.032..0.133 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.506..0.507 rows=962 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                        Buffers: shared hit=21
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.040..0.276 rows=962 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=21
                          ->  Limit  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.161..0.165 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.160..0.164 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.73 rows=1 width=1392) (actual time=0.155..0.159 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=1384) (actual time=0.151..0.154 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.096..0.118 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.072..0.076 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.059..0.066 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.045..0.046 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.029..0.030 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.012..0.012 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                        Filter: (value = ANY ('{p025a,p025b}'::text[]))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.228 ms
Execution Time: 11.825 ms
```

### text in (2 values) 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=9247.84..9247.85 rows=1 width=8) (actual time=10.042..10.045 rows=1 loops=1)
  Buffers: shared hit=3869
  ->  Limit  (cost=88.56..9244.26 rows=287 width=4) (actual time=0.548..9.828 rows=962 loops=1)
        Buffers: shared hit=3869
        ->  Result  (cost=88.56..9244.26 rows=287 width=4) (actual time=0.547..9.534 rows=962 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=3869
              ->  Nested Loop  (cost=88.56..9244.26 rows=287 width=0) (actual time=0.540..9.234 rows=962 loops=1)
                    Buffers: shared hit=3869
                    ->  HashAggregate  (cost=88.12..99.41 rows=1128 width=32) (actual time=0.521..0.923 rows=962 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=21
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.036..0.226 rows=962 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                Heap Fetches: 0
                                Buffers: shared hit=21
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.11 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=962)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=3848
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.554 ms
Execution Time: 10.108 ms
```

### text in (2 values) 0.25%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=232.81..232.94 rows=51 width=1426) (actual time=15.252..15.368 rows=51 loops=1)
  Buffers: shared hit=6758
  ->  Sort  (cost=232.81..233.53 rows=287 width=1426) (actual time=15.249..15.352 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 191kB
        Buffers: shared hit=6758
        ->  Result  (cost=89.12..223.24 rows=287 width=1426) (actual time=0.971..14.164 rows=962 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=6758
              ->  Nested Loop Left Join  (cost=89.12..221.09 rows=287 width=1399) (actual time=0.948..13.321 rows=962 loops=1)
                    Buffers: shared hit=6758
                    ->  Nested Loop  (cost=88.82..127.17 rows=287 width=1391) (actual time=0.925..9.180 rows=962 loops=1)
                          Buffers: shared hit=3872
                          ->  Nested Loop  (cost=88.40..118.98 rows=1 width=71) (actual time=0.896..1.493 rows=962 loops=1)
                                Buffers: shared hit=24
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.032..0.038 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=88.12..99.41 rows=1128 width=32) (actual time=0.861..1.208 rows=962 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=21
                                      ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.035..0.341 rows=962 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=21
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.11 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=962)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3848
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=962)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=2886
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=962)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=2886
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.759 ms
Execution Time: 15.528 ms
```

### text in (2 values) 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1265.23..1265.24 rows=1 width=8) (actual time=32.525..32.547 rows=1 loops=1)
  Buffers: shared hit=16335
  ->  Limit  (cost=97.15..1265.08 rows=12 width=4) (actual time=32.521..32.542 rows=0 loops=1)
        Buffers: shared hit=16335
        ->  Result  (cost=97.15..1265.08 rows=12 width=4) (actual time=32.518..32.539 rows=0 loops=1)
              Buffers: shared hit=16335
              ->  Append  (cost=97.15..1264.96 rows=12 width=0) (actual time=32.517..32.538 rows=0 loops=1)
                    Buffers: shared hit=16335
                    ->  Subquery Scan on "*SELECT* 1"  (cost=97.15..652.94 rows=9 width=0) (actual time=28.541..28.548 rows=0 loops=1)
                          Buffers: shared hit=15690
                          ->  Limit  (cost=97.15..652.85 rows=9 width=4) (actual time=28.540..28.546 rows=0 loops=1)
                                Buffers: shared hit=15690
                                ->  Result  (cost=97.15..652.85 rows=9 width=4) (actual time=28.538..28.543 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15690
                                      ->  Hash Semi Join  (cost=97.15..652.85 rows=9 width=0) (actual time=28.522..28.526 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15690
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.051..21.762 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.081..0.083 rows=43 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                  Buffers: shared hit=9
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.038..0.062 rows=43 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=9
                    ->  Subquery Scan on "*SELECT* 2"  (cost=97.02..432.68 rows=1 width=0) (actual time=3.355..3.359 rows=0 loops=1)
                          Buffers: shared hit=532
                          ->  Limit  (cost=97.02..432.67 rows=1 width=4) (actual time=3.352..3.356 rows=0 loops=1)
                                Buffers: shared hit=532
                                ->  Result  (cost=97.02..432.67 rows=1 width=4) (actual time=3.349..3.352 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=532
                                      ->  Hash Semi Join  (cost=97.02..432.67 rows=1 width=0) (actual time=3.328..3.330 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=532
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.055..2.906 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.075..0.076 rows=43 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                  Buffers: shared hit=9
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.043..0.060 rows=43 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=9
                    ->  Subquery Scan on "*SELECT* 3"  (cost=97.02..133.54 rows=1 width=0) (actual time=0.408..0.411 rows=0 loops=1)
                          Buffers: shared hit=76
                          ->  Limit  (cost=97.02..133.53 rows=1 width=4) (actual time=0.407..0.409 rows=0 loops=1)
                                Buffers: shared hit=76
                                ->  Result  (cost=97.02..133.53 rows=1 width=4) (actual time=0.405..0.406 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=76
                                      ->  Hash Semi Join  (cost=97.02..133.53 rows=1 width=0) (actual time=0.385..0.386 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=76
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.047..0.212 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.089..0.090 rows=43 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                  Buffers: shared hit=9
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.038..0.066 rows=43 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=9
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.74 rows=1 width=0) (actual time=0.204..0.209 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.73 rows=1 width=4) (actual time=0.203..0.207 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.73 rows=1 width=4) (actual time=0.201..0.205 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=0) (actual time=0.191..0.194 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.109..0.141 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.084..0.088 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.061..0.071 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.047..0.048 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.020..0.020 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.025..0.025 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.014..0.015 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.016..0.016 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                  Filter: (value = ANY ('{p001a,p001b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.037 ms
Execution Time: 33.040 ms
```

### text in (2 values) 0.01%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=9867.22..9883.77 rows=1 width=1434) (actual time=3.076..3.101 rows=0 loops=1)
  Buffers: shared hit=826
  ->  Nested Loop  (cost=9866.92..9875.44 rows=1 width=1399) (actual time=3.075..3.098 rows=0 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=826
        ->  Subquery Scan on x  (cost=9866.64..9867.12 rows=1 width=1392) (actual time=3.074..3.096 rows=0 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=826
              ->  Limit  (cost=9866.64..9866.85 rows=12 width=1392) (actual time=3.072..3.094 rows=0 loops=1)
                    Buffers: shared hit=826
                    ->  Merge Append  (cost=9866.64..9866.85 rows=12 width=1392) (actual time=3.070..3.091 rows=0 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=826
                          ->  Limit  (cost=9254.63..9254.65 rows=9 width=1392) (actual time=0.735..0.740 rows=0 loops=1)
                                Buffers: shared hit=181
                                ->  Sort  (cost=9254.63..9254.65 rows=9 width=1392) (actual time=0.733..0.738 rows=0 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=181
                                      ->  Result  (cost=88.56..9254.49 rows=9 width=1392) (actual time=0.726..0.730 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=181
                                            ->  Nested Loop  (cost=88.56..9254.49 rows=9 width=1384) (actual time=0.715..0.718 rows=0 loops=1)
                                                  Buffers: shared hit=181
                                                  ->  HashAggregate  (cost=88.12..99.41 rows=1128 width=32) (actual time=0.161..0.192 rows=43 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 73kB
                                                        Buffers: shared hit=9
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.053..0.081 rows=43 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=9
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.12 rows=1 width=1384) (actual time=0.012..0.012 rows=0 loops=43)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=172
                          ->  Limit  (cost=432.68..432.69 rows=1 width=1392) (actual time=1.713..1.719 rows=0 loops=1)
                                Buffers: shared hit=532
                                ->  Sort  (cost=432.68..432.69 rows=1 width=1392) (actual time=1.710..1.715 rows=0 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=532
                                      ->  Result  (cost=97.02..432.67 rows=1 width=1392) (actual time=1.697..1.701 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=532
                                            ->  Hash Semi Join  (cost=97.02..432.67 rows=1 width=1384) (actual time=1.685..1.688 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=532
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.028..1.430 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.069..0.070 rows=43 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                        Buffers: shared hit=9
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.028..0.051 rows=43 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=9
                          ->  Limit  (cost=133.54..133.54 rows=1 width=1392) (actual time=0.353..0.356 rows=0 loops=1)
                                Buffers: shared hit=76
                                ->  Sort  (cost=133.54..133.54 rows=1 width=1392) (actual time=0.350..0.353 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=76
                                      ->  Result  (cost=97.02..133.53 rows=1 width=1392) (actual time=0.336..0.338 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=76
                                            ->  Hash Semi Join  (cost=97.02..133.53 rows=1 width=1384) (actual time=0.321..0.323 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=76
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.035..0.222 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=82.48..82.48 rows=1128 width=32) (actual time=0.071..0.072 rows=43 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                        Buffers: shared hit=9
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.031..0.053 rows=43 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=9
                          ->  Limit  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.262..0.267 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.74..45.75 rows=1 width=1392) (actual time=0.261..0.265 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.73 rows=1 width=1392) (actual time=0.238..0.242 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.73 rows=1 width=1384) (actual time=0.230..0.234 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.130..0.162 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.097..0.101 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.073..0.083 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.051..0.052 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.025..0.025 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.017..0.017 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.26 rows=1 width=32) (actual time=0.022..0.022 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid))
                                                        Filter: (value = ANY ('{p001a,p001b}'::text[]))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (never executed)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (never executed)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.224 ms
Execution Time: 3.559 ms
```

### text in (2 values) 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=9247.84..9247.85 rows=1 width=8) (actual time=1.015..1.018 rows=1 loops=1)
  Buffers: shared hit=181
  ->  Limit  (cost=88.56..9244.26 rows=287 width=4) (actual time=0.141..0.999 rows=43 loops=1)
        Buffers: shared hit=181
        ->  Result  (cost=88.56..9244.26 rows=287 width=4) (actual time=0.139..0.985 rows=43 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=181
              ->  Nested Loop  (cost=88.56..9244.26 rows=287 width=0) (actual time=0.129..0.959 rows=43 loops=1)
                    Buffers: shared hit=181
                    ->  HashAggregate  (cost=88.12..99.41 rows=1128 width=32) (actual time=0.103..0.140 rows=43 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 73kB
                          Buffers: shared hit=9
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.051..0.072 rows=43 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                Heap Fetches: 0
                                Buffers: shared hit=9
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.11 rows=1 width=32) (actual time=0.018..0.018 rows=1 loops=43)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=172
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.090 ms
Execution Time: 1.112 ms
```

### text in (2 values) 0.01%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=232.81..232.94 rows=51 width=1426) (actual time=1.093..1.111 rows=43 loops=1)
  Buffers: shared hit=313
  ->  Sort  (cost=232.81..233.53 rows=287 width=1426) (actual time=1.090..1.101 rows=43 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 111kB
        Buffers: shared hit=313
        ->  Result  (cost=89.12..223.24 rows=287 width=1426) (actual time=0.196..1.014 rows=43 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=313
              ->  Nested Loop Left Join  (cost=89.12..221.09 rows=287 width=1399) (actual time=0.174..0.955 rows=43 loops=1)
                    Buffers: shared hit=313
                    ->  Nested Loop  (cost=88.82..127.17 rows=287 width=1391) (actual time=0.151..0.681 rows=43 loops=1)
                          Buffers: shared hit=184
                          ->  Nested Loop  (cost=88.40..118.98 rows=1 width=71) (actual time=0.123..0.169 rows=43 loops=1)
                                Buffers: shared hit=12
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.032..0.035 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=88.12..99.41 rows=1128 width=32) (actual time=0.089..0.119 rows=43 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 73kB
                                      Buffers: shared hit=9
                                      ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..82.48 rows=1128 width=32) (actual time=0.035..0.053 rows=43 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '5b332fff-79ac-49b2-8294-02ddb77ece8a'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=9
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.11 rows=1 width=1384) (actual time=0.011..0.011 rows=1 loops=43)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=172
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=43)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=129
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=43)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=129
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.624 ms
Execution Time: 1.277 ms
```

### num lt 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17561.56..17561.57 rows=1 width=8) (actual time=117.683..117.696 rows=1 loops=1)
  Buffers: shared hit=5138, temp written=523
  I/O Timings: temp write=7.467
  ->  Limit  (cost=14554.84..17549.04 rows=1001 width=4) (actual time=103.898..116.422 rows=1001 loops=1)
        Buffers: shared hit=5138, temp written=523
        I/O Timings: temp write=7.467
        ->  Result  (cost=14554.84..17638.78 rows=1031 width=4) (actual time=103.897..116.053 rows=1001 loops=1)
              Buffers: shared hit=5138, temp written=523
              I/O Timings: temp write=7.467
              ->  Append  (cost=14554.84..17628.47 rows=1031 width=0) (actual time=103.895..115.700 rows=1001 loops=1)
                    Buffers: shared hit=5138, temp written=523
                    I/O Timings: temp write=7.467
                    ->  Subquery Scan on "*SELECT* 1"  (cost=14554.84..15986.78 rows=1001 width=0) (actual time=103.893..115.469 rows=1001 loops=1)
                          Buffers: shared hit=5138, temp written=523
                          I/O Timings: temp write=7.467
                          ->  Limit  (cost=14554.84..15976.77 rows=1001 width=4) (actual time=103.892..115.251 rows=1001 loops=1)
                                Buffers: shared hit=5138, temp written=523
                                I/O Timings: temp write=7.467
                                ->  Result  (cost=14554.84..16309.17 rows=1235 width=4) (actual time=103.891..115.070 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=5138, temp written=523
                                      I/O Timings: temp write=7.467
                                      ->  Hash Semi Join  (cost=14554.84..16309.17 rows=1235 width=0) (actual time=103.881..114.698 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=5138, temp written=523
                                            I/O Timings: temp write=7.467
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.038..4.216 rows=4910 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=3195
                                            ->  Hash  (cost=11447.12..11447.12 rows=160653 width=32) (actual time=102.880..102.882 rows=160258 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 6033kB
                                                  Buffers: shared hit=1943, temp written=508
                                                  I/O Timings: temp write=5.961
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..11447.12 rows=160653 width=32) (actual time=0.045..43.871 rows=160258 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '4000'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1943
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1444.26 rows=26 width=0) (never executed)
                          ->  Limit  (cost=0.86..1444.00 rows=26 width=4) (never executed)
                                ->  Result  (cost=0.86..1444.00 rows=26 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1444.00 rows=26 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '4000'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..146.92 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.86..146.89 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.86..146.89 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..146.89 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '4000'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.35 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.34 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.34 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '4000'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.265 ms
Execution Time: 119.365 ms
```

### num lt 40%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=48.56..10275.28 rows=1 width=1434) (actual time=0.804..6.396 rows=51 loops=1)
  Buffers: shared hit=2396
  ->  Nested Loop  (cost=48.26..10266.94 rows=1 width=1399) (actual time=0.774..6.005 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2243
        ->  Subquery Scan on x  (cost=47.98..10258.63 rows=1 width=1392) (actual time=0.755..5.739 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2090
              ->  Limit  (cost=47.98..10257.48 rows=51 width=1392) (actual time=0.750..5.560 rows=51 loops=1)
                    Buffers: shared hit=2090
                    ->  Merge Append  (cost=47.98..16263.07 rows=81 width=1392) (actual time=0.747..5.543 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2090
                          ->  Limit  (cost=0.86..14625.57 rows=51 width=1392) (actual time=0.240..4.992 rows=51 loops=1)
                                Buffers: shared hit=2012
                                ->  Result  (cost=0.86..354148.11 rows=1235 width=1392) (actual time=0.240..4.978 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2012
                                      ->  Nested Loop Semi Join  (cost=0.86..354148.11 rows=1235 width=1384) (actual time=0.232..4.939 rows=51 loops=1)
                                            Buffers: shared hit=2012
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..314238.64 rows=6151 width=1384) (actual time=0.101..3.965 rows=102 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1482
                                                  Buffers: shared hit=1604
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..6.48 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=102)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '4000'::double precision)
                                                  Rows Removed by Filter: 0
                                                  Buffers: shared hit=408
                          ->  Limit  (cost=0.86..1444.00 rows=26 width=1392) (actual time=0.238..0.241 rows=1 loops=1)
                                Buffers: shared hit=18
                                ->  Result  (cost=0.86..1444.00 rows=26 width=1392) (actual time=0.237..0.240 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=18
                                      ->  Nested Loop Semi Join  (cost=0.86..1444.00 rows=26 width=1384) (actual time=0.233..0.236 rows=1 loops=1)
                                            Buffers: shared hit=18
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.031..0.042 rows=3 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=6
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (actual time=0.061..0.061 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '4000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
                          ->  Limit  (cost=0.86..146.89 rows=3 width=1392) (actual time=0.096..0.099 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.86..146.89 rows=3 width=1392) (actual time=0.094..0.097 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=23
                                      ->  Nested Loop Semi Join  (cost=0.86..146.89 rows=3 width=1384) (actual time=0.089..0.091 rows=1 loops=1)
                                            Buffers: shared hit=23
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.030..0.040 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=7
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '4000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=16
                          ->  Limit  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.170..0.177 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.169..0.176 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.34 rows=1 width=1392) (actual time=0.109..0.158 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=1384) (actual time=0.106..0.153 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.085..0.112 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.062..0.067 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.045..0.055 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.033..0.035 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.013..0.013 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.011..0.012 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                        Filter: (value < '4000'::double precision)
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.527 ms
Execution Time: 6.829 ms
```

### num lt 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=4496.82..4496.83 rows=1 width=8) (actual time=16.165..16.168 rows=1 loops=1)
  Buffers: shared hit=11820
  ->  Limit  (cost=0.86..4484.31 rows=1001 width=4) (actual time=0.097..15.968 rows=1001 loops=1)
        Buffers: shared hit=11820
        ->  Result  (cost=0.86..182778.68 rows=40808 width=4) (actual time=0.095..15.766 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=11820
              ->  Nested Loop Semi Join  (cost=0.86..182778.68 rows=40808 width=0) (actual time=0.084..15.509 rows=1001 loops=1)
                    Buffers: shared hit=11820
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.042..1.587 rows=2541 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1656
                    ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2541)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                          Filter: (value < '4000'::double precision)
                          Rows Removed by Filter: 1
                          Buffers: shared hit=10164
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.815 ms
Execution Time: 16.230 ms
```

### num lt 40%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.42..619.43 rows=51 width=1426) (actual time=0.102..1.275 rows=51 loops=1)
  Buffers: shared hit=800
  ->  Result  (cost=1.42..494500.64 rows=40808 width=1426) (actual time=0.100..1.265 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=800
        ->  Nested Loop Left Join  (cost=1.42..494194.58 rows=40808 width=1399) (actual time=0.086..1.218 rows=51 loops=1)
              Buffers: shared hit=800
              ->  Nested Loop  (cost=1.12..480842.90 rows=40808 width=1391) (actual time=0.073..1.016 rows=51 loops=1)
                    Buffers: shared hit=647
                    ->  Nested Loop Semi Join  (cost=0.85..480324.50 rows=40808 width=1384) (actual time=0.061..0.970 rows=51 loops=1)
                          Buffers: shared hit=644
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.029..0.309 rows=128 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                Buffers: shared hit=132
                          ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=128)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                Filter: (value < '4000'::double precision)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=512
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.008..0.009 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.879 ms
Execution Time: 1.351 ms
```

### num lt 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2472.82..2472.83 rows=1 width=8) (actual time=31.972..31.988 rows=1 loops=1)
  Buffers: shared hit=16752
  ->  Limit  (cost=693.14..2471.99 rows=66 width=4) (actual time=5.332..31.859 rows=487 loops=1)
        Buffers: shared hit=16752
        ->  Result  (cost=693.14..2471.99 rows=66 width=4) (actual time=5.329..31.753 rows=487 loops=1)
              Buffers: shared hit=16752
              ->  Append  (cost=693.14..2471.33 rows=66 width=0) (actual time=5.328..31.616 rows=487 loops=1)
                    Buffers: shared hit=16752
                    ->  Subquery Scan on "*SELECT* 1"  (cost=693.14..1250.06 rows=63 width=0) (actual time=5.327..24.807 rows=481 loops=1)
                          Buffers: shared hit=15781
                          ->  Limit  (cost=693.14..1249.43 rows=63 width=4) (actual time=5.325..24.687 rows=481 loops=1)
                                Buffers: shared hit=15781
                                ->  Result  (cost=693.14..1249.43 rows=63 width=4) (actual time=5.323..24.572 rows=481 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15781
                                      ->  Hash Semi Join  (cost=693.14..1249.43 rows=63 width=0) (actual time=5.307..24.434 rows=481 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15781
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.045..12.611 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=589.67..589.67 rows=8233 width=32) (actual time=5.234..5.235 rows=8098 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 635kB
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..589.67 rows=8233 width=32) (actual time=0.029..2.740 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
                    ->  Subquery Scan on "*SELECT* 2"  (cost=693.02..1028.69 rows=1 width=0) (actual time=5.149..5.890 rows=4 loops=1)
                          Buffers: shared hit=623
                          ->  Limit  (cost=693.02..1028.68 rows=1 width=4) (actual time=5.147..5.885 rows=4 loops=1)
                                Buffers: shared hit=623
                                ->  Result  (cost=693.02..1028.68 rows=1 width=4) (actual time=5.144..5.880 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=623
                                      ->  Hash Semi Join  (cost=693.02..1028.68 rows=1 width=0) (actual time=5.124..5.858 rows=4 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=623
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.038..0.945 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=589.67..589.67 rows=8233 width=32) (actual time=4.759..4.760 rows=8098 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 635kB
                                                  Buffers: shared hit=100
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..589.67 rows=8233 width=32) (actual time=0.030..2.415 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=100
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..146.90 rows=1 width=0) (actual time=0.437..0.671 rows=2 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..146.89 rows=1 width=4) (actual time=0.436..0.669 rows=2 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..146.89 rows=1 width=4) (actual time=0.434..0.665 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..146.89 rows=1 width=0) (actual time=0.418..0.649 rows=2 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.030..0.146 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.35 rows=1 width=0) (actual time=0.126..0.129 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.34 rows=1 width=4) (actual time=0.124..0.127 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.34 rows=1 width=4) (actual time=0.123..0.126 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=0) (actual time=0.116..0.118 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.073..0.094 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.058 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.046 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.369 ms
Execution Time: 32.286 ms
```

### num lt 2%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=16532.85..16688.87 rows=1 width=1434) (actual time=65.067..65.459 rows=51 loops=1)
  Buffers: shared hit=25742
  ->  Nested Loop  (cost=16532.55..16680.54 rows=1 width=1399) (actual time=65.042..65.274 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=25589
        ->  Subquery Scan on x  (cost=16532.27..16672.22 rows=1 width=1392) (actual time=65.026..65.131 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=25436
              ->  Limit  (cost=16532.27..16671.08 rows=51 width=1392) (actual time=65.022..65.085 rows=51 loops=1)
                    Buffers: shared hit=25436
                    ->  Merge Append  (cost=16532.27..16679.24 rows=54 width=1392) (actual time=65.020..65.076 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=25436
                          ->  Limit  (cost=15457.32..15457.45 rows=51 width=1392) (actual time=58.409..58.432 rows=51 loops=1)
                                Buffers: shared hit=24592
                                ->  Sort  (cost=15457.32..15457.48 rows=63 width=1392) (actual time=58.407..58.424 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24592
                                      ->  Result  (cost=693.14..15455.44 rows=63 width=1392) (actual time=5.371..57.660 rows=481 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=24592
                                            ->  Hash Semi Join  (cost=693.14..15455.44 rows=63 width=1384) (actual time=5.360..57.225 rows=481 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=24592
                                                  ->  Index Scan using records_list_group on records r  (cost=0.55..14746.00 rows=6151 width=1384) (actual time=0.039..43.575 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=24492
                                                  ->  Hash  (cost=589.67..589.67 rows=8233 width=32) (actual time=5.264..5.265 rows=8098 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 635kB
                                                        Buffers: shared hit=100
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..589.67 rows=8233 width=32) (actual time=0.027..2.696 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=100
                          ->  Limit  (cost=1028.69..1028.69 rows=1 width=1392) (actual time=6.010..6.012 rows=1 loops=1)
                                Buffers: shared hit=623
                                ->  Sort  (cost=1028.69..1028.69 rows=1 width=1392) (actual time=6.005..6.007 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 33kB
                                      Buffers: shared hit=623
                                      ->  Result  (cost=693.02..1028.68 rows=1 width=1392) (actual time=5.178..5.967 rows=4 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=623
                                            ->  Hash Semi Join  (cost=693.02..1028.68 rows=1 width=1384) (actual time=5.161..5.948 rows=4 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=623
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.029..0.992 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=589.67..589.67 rows=8233 width=32) (actual time=4.727..4.727 rows=8098 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 635kB
                                                        Buffers: shared hit=100
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..589.67 rows=8233 width=32) (actual time=0.032..2.276 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=100
                          ->  Limit  (cost=0.86..146.89 rows=1 width=1392) (actual time=0.432..0.434 rows=1 loops=1)
                                Buffers: shared hit=184
                                ->  Result  (cost=0.86..146.89 rows=1 width=1392) (actual time=0.430..0.431 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=184
                                      ->  Nested Loop Semi Join  (cost=0.86..146.89 rows=1 width=1384) (actual time=0.415..0.416 rows=1 loops=1)
                                            Buffers: shared hit=184
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.037..0.104 rows=36 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=40
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=36)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=144
                          ->  Limit  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.159..0.163 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.158..0.161 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.34 rows=1 width=1392) (actual time=0.145..0.148 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=1384) (actual time=0.139..0.142 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.085..0.108 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.063..0.066 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.047..0.054 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.017..0.017 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.012..0.012 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
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
Planning Time: 5.864 ms
Execution Time: 65.970 ms
```

### num lt 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=7699.11..7699.12 rows=1 width=8) (actual time=33.580..33.585 rows=1 loops=1)
  Buffers: shared hit=32363
  ->  Limit  (cost=693.02..7686.59 rows=1001 width=4) (actual time=3.830..33.400 rows=1001 loops=1)
        Buffers: shared hit=32363
        ->  Result  (cost=693.02..15301.97 rows=2091 width=4) (actual time=3.827..33.203 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32363
              ->  Hash Semi Join  (cost=693.02..15301.97 rows=2091 width=0) (actual time=3.818..32.976 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=32363
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.081..17.632 rows=49778 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=32263
                    ->  Hash  (cost=589.67..589.67 rows=8233 width=32) (actual time=3.696..3.698 rows=8098 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 635kB
                          Buffers: shared hit=100
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..589.67 rows=8233 width=32) (actual time=0.023..2.010 rows=8098 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '200'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=100
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.528 ms
Execution Time: 33.641 ms
```

### num lt 2%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1046.69..1046.82 rows=51 width=1426) (actual time=109.302..109.336 rows=51 loops=1)
  Buffers: shared hit=32549
  ->  Sort  (cost=1046.69..1051.92 rows=2091 width=1426) (actual time=109.299..109.325 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 205kB
        Buffers: shared hit=32549
        ->  Result  (cost=776.23..976.93 rows=2091 width=1426) (actual time=5.099..99.616 rows=8098 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32549
              ->  Hash Left Join  (cost=776.23..961.25 rows=2091 width=1399) (actual time=5.079..91.247 rows=8098 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32549
                    ->  Nested Loop  (cost=631.53..811.05 rows=2091 width=1391) (actual time=4.855..84.959 rows=8098 loops=1)
                          Buffers: shared hit=32495
                          ->  Nested Loop  (cost=631.11..803.79 rows=1 width=71) (actual time=4.816..11.510 rows=8098 loops=1)
                                Buffers: shared hit=103
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.024..0.028 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=630.83..713.16 rows=8233 width=32) (actual time=4.789..9.165 rows=8098 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=100
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..589.67 rows=8233 width=32) (actual time=0.022..1.704 rows=8098 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '200'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=100
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.77 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=8098)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=32392
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.209..0.212 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.047..0.190 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.036..0.164 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.017..0.017 rows=53 loops=1)
                                            Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.187 ms
Execution Time: 109.493 ms
```

### num lt 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1258.75..1258.76 rows=1 width=8) (actual time=17.462..17.475 rows=1 loops=1)
  Buffers: shared hit=16353
  ->  Limit  (cost=95.14..1258.61 rows=11 width=4) (actual time=0.808..17.448 rows=40 loops=1)
        Buffers: shared hit=16353
        ->  Result  (cost=95.14..1258.61 rows=11 width=4) (actual time=0.807..17.433 rows=40 loops=1)
              Buffers: shared hit=16353
              ->  Append  (cost=95.14..1258.50 rows=11 width=0) (actual time=0.806..17.416 rows=40 loops=1)
                    Buffers: shared hit=16353
                    ->  Subquery Scan on "*SELECT* 1"  (cost=95.14..650.89 rows=8 width=0) (actual time=0.804..14.980 rows=39 loops=1)
                          Buffers: shared hit=15696
                          ->  Limit  (cost=95.14..650.81 rows=8 width=4) (actual time=0.803..14.962 rows=39 loops=1)
                                Buffers: shared hit=15696
                                ->  Result  (cost=95.14..650.81 rows=8 width=4) (actual time=0.803..14.948 rows=39 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15696
                                      ->  Hash Semi Join  (cost=95.14..650.81 rows=8 width=0) (actual time=0.791..14.918 rows=39 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15696
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.035..9.808 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=80.95..80.95 rows=1090 width=32) (actual time=0.542..0.543 rows=1002 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 79kB
                                                  Buffers: shared hit=15
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.021..0.282 rows=1002 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15
                    ->  Subquery Scan on "*SELECT* 2"  (cost=95.01..430.67 rows=1 width=0) (actual time=1.580..1.681 rows=1 loops=1)
                          Buffers: shared hit=538
                          ->  Limit  (cost=95.01..430.66 rows=1 width=4) (actual time=1.579..1.678 rows=1 loops=1)
                                Buffers: shared hit=538
                                ->  Result  (cost=95.01..430.66 rows=1 width=4) (actual time=1.576..1.675 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=538
                                      ->  Hash Semi Join  (cost=95.01..430.66 rows=1 width=0) (actual time=1.555..1.653 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=538
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.049..0.883 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=80.95..80.95 rows=1090 width=32) (actual time=0.650..0.651 rows=1002 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 79kB
                                                  Buffers: shared hit=15
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.078..0.370 rows=1002 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15
                    ->  Subquery Scan on "*SELECT* 3"  (cost=95.01..131.53 rows=1 width=0) (actual time=0.604..0.605 rows=0 loops=1)
                          Buffers: shared hit=82
                          ->  Limit  (cost=95.01..131.52 rows=1 width=4) (actual time=0.602..0.604 rows=0 loops=1)
                                Buffers: shared hit=82
                                ->  Result  (cost=95.01..131.52 rows=1 width=4) (actual time=0.601..0.602 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=82
                                      ->  Hash Semi Join  (cost=95.01..131.52 rows=1 width=0) (actual time=0.590..0.591 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=82
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.023..0.115 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=80.95..80.95 rows=1090 width=32) (actual time=0.457..0.458 rows=1002 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 79kB
                                                  Buffers: shared hit=15
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.018..0.249 rows=1002 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.35 rows=1 width=0) (actual time=0.123..0.126 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.34 rows=1 width=4) (actual time=0.123..0.125 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.34 rows=1 width=4) (actual time=0.122..0.125 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=0) (actual time=0.119..0.121 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.068..0.088 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.052 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.036..0.042 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '25'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.588 ms
Execution Time: 17.699 ms
```

### num lt 0.25%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=9568.34..9584.86 rows=1 width=1434) (actual time=12.093..12.472 rows=40 loops=1)
  Buffers: shared hit=4920
  ->  Nested Loop  (cost=9568.04..9576.52 rows=1 width=1399) (actual time=12.073..12.314 rows=40 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4800
        ->  Subquery Scan on x  (cost=9567.76..9568.21 rows=1 width=1392) (actual time=12.059..12.190 rows=40 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4680
              ->  Limit  (cost=9567.76..9567.96 rows=11 width=1392) (actual time=12.055..12.148 rows=40 loops=1)
                    Buffers: shared hit=4680
                    ->  Merge Append  (cost=9567.76..9567.96 rows=11 width=1392) (actual time=12.053..12.138 rows=40 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4680
                          ->  Limit  (cost=8960.17..8960.19 rows=8 width=1392) (actual time=9.316..9.340 rows=39 loops=1)
                                Buffers: shared hit=4023
                                ->  Sort  (cost=8960.17..8960.19 rows=8 width=1392) (actual time=9.314..9.333 rows=39 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 103kB
                                      Buffers: shared hit=4023
                                      ->  Result  (cost=86.84..8960.05 rows=8 width=1392) (actual time=0.625..9.221 rows=39 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=4023
                                            ->  Nested Loop  (cost=86.84..8960.05 rows=8 width=1384) (actual time=0.614..9.185 rows=39 loops=1)
                                                  Buffers: shared hit=4023
                                                  ->  HashAggregate  (cost=86.40..97.30 rows=1090 width=32) (actual time=0.586..0.970 rows=1002 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=15
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.036..0.230 rows=1002 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=15
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.13 rows=1 width=1384) (actual time=0.008..0.008 rows=0 loops=1002)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=4008
                          ->  Limit  (cost=430.67..430.68 rows=1 width=1392) (actual time=1.946..1.950 rows=1 loops=1)
                                Buffers: shared hit=538
                                ->  Sort  (cost=430.67..430.68 rows=1 width=1392) (actual time=1.942..1.945 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=538
                                      ->  Result  (cost=95.01..430.66 rows=1 width=1392) (actual time=1.764..1.919 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=538
                                            ->  Hash Semi Join  (cost=95.01..430.66 rows=1 width=1384) (actual time=1.675..1.829 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=538
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.035..1.159 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=80.95..80.95 rows=1090 width=32) (actual time=0.544..0.545 rows=1002 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 79kB
                                                        Buffers: shared hit=15
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.028..0.290 rows=1002 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=15
                          ->  Limit  (cost=131.53..131.53 rows=1 width=1392) (actual time=0.648..0.650 rows=0 loops=1)
                                Buffers: shared hit=82
                                ->  Sort  (cost=131.53..131.53 rows=1 width=1392) (actual time=0.646..0.648 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=82
                                      ->  Result  (cost=95.01..131.52 rows=1 width=1392) (actual time=0.632..0.634 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=82
                                            ->  Hash Semi Join  (cost=95.01..131.52 rows=1 width=1384) (actual time=0.620..0.621 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=82
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.029..0.136 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=80.95..80.95 rows=1090 width=32) (actual time=0.468..0.468 rows=1002 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 79kB
                                                        Buffers: shared hit=15
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.022..0.256 rows=1002 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=15
                          ->  Limit  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.134..0.138 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.133..0.136 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.34 rows=1 width=1392) (actual time=0.130..0.134 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=1384) (actual time=0.126..0.129 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.074..0.096 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.053..0.056 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.045 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.011 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.011..0.011 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                        Filter: (value < '25'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=40)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=120
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=40)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=120
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.533 ms
Execution Time: 12.858 ms
```

### num lt 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=8955.83..8955.84 rows=1 width=8) (actual time=23.454..23.460 rows=1 loops=1)
  Buffers: shared hit=4019
  ->  Limit  (cost=86.84..8952.36 rows=277 width=4) (actual time=1.165..22.676 rows=1001 loops=1)
        Buffers: shared hit=4019
        ->  Result  (cost=86.84..8952.36 rows=277 width=4) (actual time=1.162..22.084 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=4019
              ->  Nested Loop  (cost=86.84..8952.36 rows=277 width=0) (actual time=1.148..21.246 rows=1001 loops=1)
                    Buffers: shared hit=4019
                    ->  HashAggregate  (cost=86.40..97.30 rows=1090 width=32) (actual time=1.069..1.682 rows=1001 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=15
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.093..0.485 rows=1002 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=15
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.12 rows=1 width=32) (actual time=0.018..0.018 rows=1 loops=1001)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=4004
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.004 ms
Execution Time: 23.587 ms
```

### num lt 0.25%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=226.66..226.78 rows=51 width=1426) (actual time=40.489..40.528 rows=51 loops=1)
  Buffers: shared hit=7032
  ->  Sort  (cost=226.66..227.35 rows=277 width=1426) (actual time=40.485..40.513 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 201kB
        Buffers: shared hit=7032
        ->  Result  (cost=87.40..217.42 rows=277 width=1426) (actual time=1.218..37.687 rows=1002 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=7032
              ->  Nested Loop Left Join  (cost=87.40..215.34 rows=277 width=1399) (actual time=1.189..35.464 rows=1002 loops=1)
                    Buffers: shared hit=7032
                    ->  Nested Loop  (cost=87.10..124.70 rows=277 width=1391) (actual time=1.142..22.131 rows=1002 loops=1)
                          Buffers: shared hit=4026
                          ->  Nested Loop  (cost=86.67..116.49 rows=1 width=71) (actual time=1.051..2.481 rows=1002 loops=1)
                                Buffers: shared hit=18
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.042..0.047 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=86.40..97.30 rows=1090 width=32) (actual time=1.005..1.917 rows=1002 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=15
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..80.95 rows=1090 width=32) (actual time=0.040..0.440 rows=1002 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '25'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=15
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.12 rows=1 width=1384) (actual time=0.018..0.018 rows=1 loops=1002)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=4008
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=1002)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=3006
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=1002)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=3006
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.758 ms
Execution Time: 40.732 ms
```

### num lt 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=808.89..808.90 rows=1 width=8) (actual time=5.960..5.988 rows=1 loops=1)
  Buffers: shared hit=835
  ->  Limit  (cost=6.07..808.84 rows=4 width=4) (actual time=0.784..5.975 rows=3 loops=1)
        Buffers: shared hit=835
        ->  Result  (cost=6.07..808.84 rows=4 width=4) (actual time=0.781..5.969 rows=3 loops=1)
              Buffers: shared hit=835
              ->  Append  (cost=6.07..808.80 rows=4 width=0) (actual time=0.780..5.965 rows=3 loops=1)
                    Buffers: shared hit=835
                    ->  Subquery Scan on "*SELECT* 1"  (cost=6.07..378.45 rows=1 width=0) (actual time=0.778..1.622 rows=3 loops=1)
                          Buffers: shared hit=200
                          ->  Limit  (cost=6.07..378.44 rows=1 width=4) (actual time=0.776..1.617 rows=3 loops=1)
                                Buffers: shared hit=200
                                ->  Result  (cost=6.07..378.44 rows=1 width=4) (actual time=0.775..1.613 rows=3 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=200
                                      ->  Nested Loop  (cost=6.07..378.44 rows=1 width=0) (actual time=0.758..1.594 rows=3 loops=1)
                                            Buffers: shared hit=200
                                            ->  HashAggregate  (cost=5.63..6.08 rows=44 width=32) (actual time=0.213..0.250 rows=49 loops=1)
                                                  Group Key: v.tenant_id, v.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.42 rows=44 width=32) (actual time=0.158..0.173 rows=49 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.026..0.026 rows=0 loops=49)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=196
                    ->  Subquery Scan on "*SELECT* 2"  (cost=6.40..342.06 rows=1 width=0) (actual time=2.302..2.308 rows=0 loops=1)
                          Buffers: shared hit=527
                          ->  Limit  (cost=6.40..342.05 rows=1 width=4) (actual time=2.301..2.306 rows=0 loops=1)
                                Buffers: shared hit=527
                                ->  Result  (cost=6.40..342.05 rows=1 width=4) (actual time=2.297..2.302 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=527
                                      ->  Hash Semi Join  (cost=6.40..342.05 rows=1 width=0) (actual time=2.283..2.286 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=527
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.031..1.889 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Hash  (cost=5.42..5.42 rows=44 width=32) (actual time=0.248..0.249 rows=49 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 12kB
                                                  Buffers: shared hit=4
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..5.42 rows=44 width=32) (actual time=0.025..0.050 rows=49 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 3"  (cost=6.40..42.92 rows=1 width=0) (actual time=1.508..1.512 rows=0 loops=1)
                          Buffers: shared hit=71
                          ->  Limit  (cost=6.40..42.91 rows=1 width=4) (actual time=1.505..1.509 rows=0 loops=1)
                                Buffers: shared hit=71
                                ->  Result  (cost=6.40..42.91 rows=1 width=4) (actual time=1.502..1.505 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=71
                                      ->  Hash Semi Join  (cost=6.40..42.91 rows=1 width=0) (actual time=1.484..1.487 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=71
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.045..0.332 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=5.42..5.42 rows=44 width=32) (actual time=1.113..1.114 rows=49 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 12kB
                                                  Buffers: shared hit=4
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..5.42 rows=44 width=32) (actual time=0.291..1.028 rows=49 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.35 rows=1 width=0) (actual time=0.502..0.508 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.34 rows=1 width=4) (actual time=0.456..0.461 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.34 rows=1 width=4) (actual time=0.453..0.458 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=0) (actual time=0.437..0.441 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.235..0.367 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.128..0.139 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.103..0.115 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.085..0.087 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.058..0.059 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.047..0.047 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.021..0.021 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                  Filter: (value < '1'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 5.761 ms
Execution Time: 6.433 ms
```

### num lt 0.01%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=809.44..825.61 rows=1 width=1434) (actual time=6.566..6.632 rows=3 loops=1)
  Buffers: shared hit=853
  ->  Nested Loop  (cost=809.14..817.27 rows=1 width=1399) (actual time=6.531..6.578 rows=3 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=844
        ->  Subquery Scan on x  (cost=808.86..808.96 rows=1 width=1392) (actual time=6.506..6.540 rows=3 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=835
              ->  Limit  (cost=808.86..808.87 rows=4 width=1392) (actual time=6.498..6.526 rows=3 loops=1)
                    Buffers: shared hit=835
                    ->  Sort  (cost=808.86..808.87 rows=4 width=1392) (actual time=6.494..6.520 rows=3 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Sort Method: quicksort  Memory: 31kB
                          Buffers: shared hit=835
                          ->  Append  (cost=378.45..808.82 rows=4 width=1392) (actual time=1.515..6.496 rows=3 loops=1)
                                Buffers: shared hit=835
                                ->  Limit  (cost=378.45..378.45 rows=1 width=1392) (actual time=1.513..1.521 rows=3 loops=1)
                                      Buffers: shared hit=200
                                      ->  Sort  (cost=378.45..378.45 rows=1 width=1392) (actual time=1.511..1.517 rows=3 loops=1)
                                            Sort Key: r.updated_at DESC, r.id DESC
                                            Sort Method: quicksort  Memory: 31kB
                                            Buffers: shared hit=200
                                            ->  Result  (cost=6.07..378.44 rows=1 width=1392) (actual time=0.421..1.491 rows=3 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=200
                                                  ->  Nested Loop  (cost=6.07..378.44 rows=1 width=1384) (actual time=0.402..1.466 rows=3 loops=1)
                                                        Buffers: shared hit=200
                                                        ->  HashAggregate  (cost=5.63..6.08 rows=44 width=32) (actual time=0.105..0.130 rows=49 loops=1)
                                                              Group Key: v.tenant_id, v.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.42 rows=44 width=32) (actual time=0.053..0.067 rows=49 loops=1)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.026..0.026 rows=0 loops=49)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                              Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=196
                                ->  Limit  (cost=342.06..342.07 rows=1 width=1392) (actual time=3.567..3.574 rows=0 loops=1)
                                      Buffers: shared hit=527
                                      ->  Sort  (cost=342.06..342.07 rows=1 width=1392) (actual time=3.564..3.569 rows=0 loops=1)
                                            Sort Key: r_1.updated_at DESC, r_1.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=527
                                            ->  Result  (cost=6.40..342.05 rows=1 width=1392) (actual time=3.546..3.551 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=527
                                                  ->  Hash Semi Join  (cost=6.40..342.05 rows=1 width=1384) (actual time=3.532..3.536 rows=0 loops=1)
                                                        Hash Cond: (r_1.id = v_1.record_id)
                                                        Buffers: shared hit=527
                                                        ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.030..3.179 rows=484 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Rows Removed by Filter: 29
                                                              Buffers: shared hit=523
                                                        ->  Hash  (cost=5.42..5.42 rows=44 width=32) (actual time=0.098..0.100 rows=49 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 12kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..5.42 rows=44 width=32) (actual time=0.070..0.081 rows=49 loops=1)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                ->  Limit  (cost=42.92..42.92 rows=1 width=1392) (actual time=1.088..1.092 rows=0 loops=1)
                                      Buffers: shared hit=71
                                      ->  Sort  (cost=42.92..42.92 rows=1 width=1392) (actual time=1.084..1.087 rows=0 loops=1)
                                            Sort Key: r_2.updated_at DESC, r_2.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=71
                                            ->  Result  (cost=6.40..42.91 rows=1 width=1392) (actual time=1.029..1.032 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=71
                                                  ->  Hash Semi Join  (cost=6.40..42.91 rows=1 width=1384) (actual time=0.941..0.943 rows=0 loops=1)
                                                        Hash Cond: (r_2.id = v_2.record_id)
                                                        Buffers: shared hit=71
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.258..0.796 rows=61 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 3
                                                              Buffers: shared hit=67
                                                        ->  Hash  (cost=5.42..5.42 rows=44 width=32) (actual time=0.096..0.097 rows=49 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 12kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..5.42 rows=44 width=32) (actual time=0.049..0.065 rows=49 loops=1)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                ->  Limit  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.286..0.291 rows=0 loops=1)
                                      Buffers: shared hit=37
                                      ->  Sort  (cost=45.35..45.36 rows=1 width=1392) (actual time=0.284..0.288 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=37
                                            ->  Result  (cost=36.85..45.34 rows=1 width=1392) (actual time=0.239..0.243 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop Semi Join  (cost=36.85..45.34 rows=1 width=1384) (actual time=0.222..0.226 rows=0 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.133..0.170 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.098..0.103 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.072..0.083 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.053..0.054 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.027..0.028 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.024..0.024 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.018..0.018 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.87 rows=1 width=32) (actual time=0.016..0.016 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid))
                                                              Filter: (value < '1'::double precision)
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.007..0.007 rows=1 loops=3)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=9
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=3)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=9
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 8.171 ms
Execution Time: 7.286 ms
```

### num lt 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=378.35..378.36 rows=1 width=8) (actual time=0.942..0.947 rows=1 loops=1)
  Buffers: shared hit=200
  ->  Limit  (cost=6.07..378.22 rows=11 width=4) (actual time=0.164..0.927 rows=49 loops=1)
        Buffers: shared hit=200
        ->  Result  (cost=6.07..378.22 rows=11 width=4) (actual time=0.160..0.912 rows=49 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=200
              ->  Nested Loop  (cost=6.07..378.22 rows=11 width=0) (actual time=0.143..0.882 rows=49 loops=1)
                    Buffers: shared hit=200
                    ->  HashAggregate  (cost=5.63..6.08 rows=44 width=32) (actual time=0.104..0.124 rows=49 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=4
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.42 rows=44 width=32) (actual time=0.062..0.074 rows=49 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=4
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.015..0.015 rows=1 loops=49)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=196
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.782 ms
Execution Time: 1.080 ms
```

### num lt 0.01%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=27.14..27.17 rows=11 width=1426) (actual time=1.084..1.100 rows=49 loops=1)
  Buffers: shared hit=350
  ->  Sort  (cost=27.14..27.17 rows=11 width=1426) (actual time=1.083..1.092 rows=49 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 123kB
        Buffers: shared hit=350
        ->  Result  (cost=6.63..26.95 rows=11 width=1426) (actual time=0.235..1.012 rows=49 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=350
              ->  Nested Loop Left Join  (cost=6.63..26.87 rows=11 width=1399) (actual time=0.163..0.905 rows=49 loops=1)
                    Buffers: shared hit=350
                    ->  Nested Loop  (cost=6.33..23.26 rows=11 width=1391) (actual time=0.143..0.667 rows=49 loops=1)
                          Buffers: shared hit=203
                          ->  Nested Loop  (cost=5.91..14.81 rows=1 width=71) (actual time=0.121..0.148 rows=49 loops=1)
                                Buffers: shared hit=7
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.054..0.056 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=5.63..6.08 rows=44 width=32) (actual time=0.065..0.078 rows=49 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 24kB
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.42 rows=44 width=32) (actual time=0.027..0.038 rows=49 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'c9c121a6-40b0-4b0b-933f-b96b2a633059'::uuid) AND (value < '1'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=49)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=196
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=49)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=147
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=49)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=147
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.508 ms
Execution Time: 1.237 ms
```

### ts range 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=31315.57..31315.58 rows=1 width=8) (actual time=195.847..195.862 rows=1 loops=1)
  Buffers: shared hit=16653, temp written=1374
  I/O Timings: temp write=15.124
  ->  Limit  (cost=25459.37..31305.83 rows=779 width=4) (actual time=166.226..195.581 rows=1001 loops=1)
        Buffers: shared hit=16653, temp written=1374
        I/O Timings: temp write=15.124
        ->  Result  (cost=25459.37..31305.83 rows=779 width=4) (actual time=166.225..195.388 rows=1001 loops=1)
              Buffers: shared hit=16653, temp written=1374
              I/O Timings: temp write=15.124
              ->  Append  (cost=25459.37..31298.04 rows=779 width=0) (actual time=166.224..195.129 rows=1001 loops=1)
                    Buffers: shared hit=16653, temp written=1374
                    I/O Timings: temp write=15.124
                    ->  Subquery Scan on "*SELECT* 1"  (cost=25459.37..29620.35 rows=760 width=0) (actual time=166.222..194.873 rows=1001 loops=1)
                          Buffers: shared hit=16653, temp written=1374
                          I/O Timings: temp write=15.124
                          ->  Limit  (cost=25459.37..29612.75 rows=760 width=4) (actual time=166.221..194.533 rows=1001 loops=1)
                                Buffers: shared hit=16653, temp written=1374
                                I/O Timings: temp write=15.124
                                ->  Result  (cost=25459.37..29612.75 rows=760 width=4) (actual time=166.220..194.215 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=16653, temp written=1374
                                      I/O Timings: temp write=15.124
                                      ->  Nested Loop Semi Join  (cost=25459.37..29612.75 rows=760 width=0) (actual time=166.207..193.917 rows=1001 loops=1)
                                            Buffers: shared hit=16653, temp written=1374
                                            I/O Timings: temp write=15.124
                                            ->  Hash Semi Join  (cost=25458.94..28042.51 rows=2156 width=48) (actual time=166.043..177.831 rows=1739 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=9697, temp written=1374
                                                  I/O Timings: temp write=15.124
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.048..5.386 rows=9650 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=6321
                                                  ->  Hash  (cost=20034.69..20034.69 rows=280456 width=32) (actual time=165.306..165.307 rows=279603 loops=1)
                                                        Buckets: 131072  Batches: 4  Memory Usage: 5387kB
                                                        Buffers: shared hit=3376, temp written=1330
                                                        I/O Timings: temp write=13.864
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..20034.69 rows=280456 width=32) (actual time=0.049..71.422 rows=279603 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=3376
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.72 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=1739)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 0
                                                  Buffers: shared hit=6956
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.29..1476.81 rows=16 width=0) (never executed)
                          ->  Limit  (cost=1.29..1476.65 rows=16 width=4) (never executed)
                                ->  Result  (cost=1.29..1476.65 rows=16 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.29..1476.65 rows=16 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=0.85..1443.14 rows=46 width=48) (never executed)
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.72 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_2.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.29..150.47 rows=2 width=0) (never executed)
                          ->  Limit  (cost=1.29..150.45 rows=2 width=4) (never executed)
                                ->  Result  (cost=1.29..150.45 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=1.29..150.45 rows=2 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=0.85..146.80 rows=5 width=48) (never executed)
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..0.72 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_4.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=37.27..46.52 rows=1 width=0) (never executed)
                          ->  Limit  (cost=37.27..46.51 rows=1 width=4) (never executed)
                                ->  Result  (cost=37.27..46.51 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=0) (never executed)
                                            ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=64) (never executed)
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 5.971 ms
Execution Time: 200.934 ms
```

### ts range 40%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $12
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=51.00..18641.97 rows=1 width=1434) (actual time=1.080..7.169 rows=51 loops=1)
  Buffers: shared hit=3279
  ->  Nested Loop  (cost=50.70..18633.64 rows=1 width=1399) (actual time=1.045..6.842 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=3126
        ->  Subquery Scan on x  (cost=50.42..18625.32 rows=1 width=1392) (actual time=1.026..6.599 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2973
              ->  Limit  (cost=50.42..18624.18 rows=51 width=1392) (actual time=1.017..6.487 rows=51 loops=1)
                    Buffers: shared hit=2973
                    ->  Merge Append  (cost=50.42..25543.81 rows=70 width=1392) (actual time=1.014..6.469 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2973
                          ->  Limit  (cost=1.29..23869.09 rows=51 width=1392) (actual time=0.223..5.630 rows=51 loops=1)
                                Buffers: shared hit=2854
                                ->  Result  (cost=1.29..355678.39 rows=760 width=1392) (actual time=0.222..5.615 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2854
                                      ->  Nested Loop Semi Join  (cost=1.29..355678.39 rows=760 width=1384) (actual time=0.206..5.569 rows=51 loops=1)
                                            Buffers: shared hit=2854
                                            ->  Nested Loop Semi Join  (cost=0.85..354108.15 rows=2156 width=1400) (actual time=0.189..5.113 rows=97 loops=1)
                                                  Buffers: shared hit=2466
                                                  ->  Index Scan using records_list_updated on records r  (cost=0.42..314238.64 rows=6151 width=1384) (actual time=0.153..3.929 rows=127 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1807
                                                        Buffers: shared hit=1958
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..6.48 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=127)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=508
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.72 rows=1 width=32) (actual time=0.004..0.004 rows=1 loops=97)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 0
                                                  Buffers: shared hit=388
                          ->  Limit  (cost=1.29..1476.65 rows=16 width=1392) (actual time=0.368..0.374 rows=1 loops=1)
                                Buffers: shared hit=40
                                ->  Result  (cost=1.29..1476.65 rows=16 width=1392) (actual time=0.367..0.372 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=40
                                      ->  Nested Loop Semi Join  (cost=1.29..1476.65 rows=16 width=1384) (actual time=0.359..0.364 rows=1 loops=1)
                                            Buffers: shared hit=40
                                            ->  Nested Loop Semi Join  (cost=0.85..1443.14 rows=46 width=1400) (actual time=0.069..0.123 rows=3 loops=1)
                                                  Buffers: shared hit=28
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.030..0.048 rows=5 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=8
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.013..0.013 rows=1 loops=5)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=20
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.72 rows=1 width=32) (actual time=0.078..0.078 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_2.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
                          ->  Limit  (cost=1.29..150.45 rows=2 width=1392) (actual time=0.118..0.123 rows=1 loops=1)
                                Buffers: shared hit=30
                                ->  Result  (cost=1.29..150.45 rows=2 width=1392) (actual time=0.116..0.120 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=30
                                      ->  Nested Loop Semi Join  (cost=1.29..150.45 rows=2 width=1384) (actual time=0.106..0.109 rows=1 loops=1)
                                            Buffers: shared hit=30
                                            ->  Nested Loop Semi Join  (cost=0.85..146.80 rows=5 width=1400) (actual time=0.054..0.085 rows=3 loops=1)
                                                  Buffers: shared hit=18
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.032..0.041 rows=3 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=6
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..8.45 rows=1 width=32) (actual time=0.013..0.013 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..0.72 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_4.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
                          ->  Limit  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.299..0.309 rows=1 loops=1)
                                Buffers: shared hit=49
                                ->  Sort  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.298..0.306 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=49
                                      ->  Result  (cost=37.27..46.51 rows=1 width=1392) (actual time=0.249..0.282 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=1384) (actual time=0.243..0.275 rows=2 loops=1)
                                                  Buffers: shared hit=49
                                                  ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=1416) (actual time=0.207..0.255 rows=3 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.184..0.211 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.150..0.154 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.132..0.141 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.115..0.117 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.075..0.076 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.037..0.037 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.016..0.016 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.013..0.013 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                              Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.028 ms
Execution Time: 7.870 ms
```

### ts range 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=9320.71..9320.72 rows=1 width=8) (actual time=26.301..26.304 rows=1 loops=1)
  Buffers: shared hit=18066
  ->  Limit  (cost=1.29..9308.20 rows=1001 width=4) (actual time=0.067..25.917 rows=1001 loops=1)
        Buffers: shared hit=18066
        ->  Result  (cost=1.29..233343.46 rows=25097 width=4) (actual time=0.066..25.699 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=18066
              ->  Nested Loop Semi Join  (cost=1.29..233343.46 rows=25097 width=0) (actual time=0.058..25.381 rows=1001 loops=1)
                    Buffers: shared hit=18066
                    ->  Nested Loop Semi Join  (cost=0.85..181458.93 rows=71240 width=48) (actual time=0.052..18.538 rows=1699 loops=1)
                          Buffers: shared hit=11270
                          ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.032..1.365 rows=2423 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                Heap Fetches: 0
                                Buffers: shared hit=1578
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=2423)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 0
                                Buffers: shared hit=9692
                    ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.72 rows=1 width=32) (actual time=0.003..0.003 rows=1 loops=1699)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                          Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                          Rows Removed by Filter: 0
                          Buffers: shared hit=6796
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.816 ms
Execution Time: 26.364 ms
```

### ts range 40%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11
```

```
Limit  (cost=1.85..1100.76 rows=51 width=1426) (actual time=0.107..1.737 rows=51 loops=1)
  Buffers: shared hit=1060
  ->  Result  (cost=1.85..540773.46 rows=25097 width=1426) (actual time=0.106..1.724 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=1060
        ->  Nested Loop Left Join  (cost=1.85..540585.23 rows=25097 width=1399) (actual time=0.090..1.669 rows=51 loops=1)
              Buffers: shared hit=1060
              ->  Nested Loop  (cost=1.55..532373.91 rows=25097 width=1391) (actual time=0.072..1.380 rows=51 loops=1)
                    Buffers: shared hit=907
                    ->  Nested Loop Semi Join  (cost=1.27..532051.91 rows=25097 width=1384) (actual time=0.057..1.323 rows=51 loops=1)
                          Buffers: shared hit=904
                          ->  Nested Loop Semi Join  (cost=0.85..480628.82 rows=71240 width=1400) (actual time=0.050..1.027 rows=85 loops=1)
                                Buffers: shared hit=564
                                ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.029..0.281 rows=112 loops=1)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Buffers: shared hit=116
                                ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=112)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                      Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                      Rows Removed by Filter: 0
                                      Buffers: shared hit=448
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.72 rows=1 width=32) (actual time=0.003..0.003 rows=1 loops=85)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value < '2025-05-01 02:40:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 0
                                Buffers: shared hit=340
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.010..0.012 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.906 ms
Execution Time: 1.834 ms
```

### ts range 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=15995.97..15995.98 rows=1 width=8) (actual time=337.074..337.094 rows=1 loops=1)
  Buffers: shared hit=52081, temp read=487 written=487
  I/O Timings: temp read=2.773 write=19.850
  ->  Limit  (cost=11746.15..15991.49 rows=358 width=4) (actual time=160.097..336.511 rows=464 loops=1)
        Buffers: shared hit=52081, temp read=487 written=487
        I/O Timings: temp read=2.773 write=19.850
        ->  Result  (cost=11746.15..15991.49 rows=358 width=4) (actual time=160.093..336.314 rows=464 loops=1)
              Buffers: shared hit=52081, temp read=487 written=487
              I/O Timings: temp read=2.773 write=19.850
              ->  Append  (cost=11746.15..15987.91 rows=358 width=0) (actual time=160.091..336.020 rows=464 loops=1)
                    Buffers: shared hit=52081, temp read=487 written=487
                    I/O Timings: temp read=2.773 write=19.850
                    ->  Subquery Scan on "*SELECT* 1"  (cost=11746.15..14325.83 rows=349 width=0) (actual time=160.089..325.597 rows=459 loops=1)
                          Buffers: shared hit=48526, temp read=487 written=487
                          I/O Timings: temp read=2.773 write=19.850
                          ->  Limit  (cost=11746.15..14322.34 rows=349 width=4) (actual time=160.087..325.329 rows=459 loops=1)
                                Buffers: shared hit=48526, temp read=487 written=487
                                I/O Timings: temp read=2.773 write=19.850
                                ->  Result  (cost=11746.15..14322.34 rows=349 width=4) (actual time=160.086..325.130 rows=459 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=48526, temp read=487 written=487
                                      I/O Timings: temp read=2.773 write=19.850
                                      ->  Nested Loop Semi Join  (cost=11746.15..14322.34 rows=349 width=0) (actual time=160.008..324.696 rows=459 loops=1)
                                            Join Filter: (r.id = v.record_id)
                                            Buffers: shared hit=48526, temp read=487 written=487
                                            I/O Timings: temp read=2.773 write=19.850
                                            ->  Hash Semi Join  (cost=11745.72..13283.37 rows=995 width=48) (actual time=159.657..231.732 rows=7820 loops=1)
                                                  Hash Cond: (r.id = v_1.record_id)
                                                  Buffers: shared hit=17246, temp read=487 written=487
                                                  I/O Timings: temp read=2.773 write=19.850
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.058..18.718 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                                  ->  Hash  (cost=9243.11..9243.11 rows=129364 width=32) (actual time=159.286..159.287 rows=128556 loops=1)
                                                        Buckets: 131072  Batches: 2  Memory Usage: 5030kB
                                                        Buffers: shared hit=1565, temp written=409
                                                        I/O Timings: temp write=15.364
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..9243.11 rows=129364 width=32) (actual time=0.052..63.538 rows=128556 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=1565
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..1.03 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=7820)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_1.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=31280
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.29..1464.90 rows=7 width=0) (actual time=0.364..8.535 rows=4 loops=1)
                          Buffers: shared hit=3127
                          ->  Limit  (cost=1.29..1464.83 rows=7 width=4) (actual time=0.363..8.529 rows=4 loops=1)
                                Buffers: shared hit=3127
                                ->  Result  (cost=1.29..1464.83 rows=7 width=4) (actual time=0.360..8.523 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=3127
                                      ->  Nested Loop Semi Join  (cost=1.29..1464.83 rows=7 width=0) (actual time=0.345..8.500 rows=4 loops=1)
                                            Join Filter: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=3127
                                            ->  Nested Loop Semi Join  (cost=0.85..1442.89 rows=21 width=48) (actual time=0.085..7.508 rows=167 loops=1)
                                                  Buffers: shared hit=2459
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.060..1.935 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..8.45 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=484)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=1936
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..1.03 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=167)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_3.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=668
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.29..148.88 rows=1 width=0) (actual time=0.297..1.389 rows=1 loops=1)
                          Buffers: shared hit=379
                          ->  Limit  (cost=1.29..148.87 rows=1 width=4) (actual time=0.296..1.386 rows=1 loops=1)
                                Buffers: shared hit=379
                                ->  Result  (cost=1.29..148.87 rows=1 width=4) (actual time=0.293..1.382 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=379
                                      ->  Nested Loop Semi Join  (cost=1.29..148.87 rows=1 width=0) (actual time=0.210..1.298 rows=1 loops=1)
                                            Join Filter: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=379
                                            ->  Nested Loop Semi Join  (cost=0.85..146.77 rows=2 width=48) (actual time=0.196..1.062 rows=17 loops=1)
                                                  Buffers: shared hit=311
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.085..0.286 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..8.45 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=61)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=244
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..1.03 rows=1 width=32) (actual time=0.013..0.013 rows=0 loops=17)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=68
                    ->  Subquery Scan on "*SELECT* 4"  (cost=37.27..46.52 rows=1 width=0) (actual time=0.215..0.219 rows=0 loops=1)
                          Buffers: shared hit=49
                          ->  Limit  (cost=37.27..46.51 rows=1 width=4) (actual time=0.214..0.217 rows=0 loops=1)
                                Buffers: shared hit=49
                                ->  Result  (cost=37.27..46.51 rows=1 width=4) (actual time=0.212..0.215 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=49
                                      ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=0) (actual time=0.199..0.202 rows=0 loops=1)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=64) (actual time=0.143..0.182 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.124..0.147 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.095..0.099 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.077..0.085 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.064..0.065 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.045..0.046 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.014..0.014 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.300 ms
Execution Time: 338.074 ms
```

### ts range 2%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $12
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22350.76..28906.96 rows=1 width=1434) (actual time=176.805..234.623 rows=51 loops=1)
  Buffers: shared hit=37638, temp read=1388 written=4110
  I/O Timings: temp read=6.920 write=80.102
  ->  Nested Loop  (cost=22350.46..28898.62 rows=1 width=1399) (actual time=176.766..234.028 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=37485, temp read=1388 written=4110
        I/O Timings: temp read=6.920 write=80.102
        ->  Subquery Scan on x  (cost=22350.18..28890.31 rows=1 width=1392) (actual time=176.741..233.577 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=37332, temp read=1388 written=4110
              I/O Timings: temp read=6.920 write=80.102
              ->  Limit  (cost=22350.18..28889.16 rows=51 width=1392) (actual time=176.733..233.301 rows=51 loops=1)
                    Buffers: shared hit=37332, temp read=1388 written=4110
                    I/O Timings: temp read=6.920 write=80.102
                    ->  Merge Append  (cost=22350.18..30043.10 rows=60 width=1392) (actual time=176.730..233.260 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=37332, temp read=1388 written=4110
                          I/O Timings: temp read=6.920 write=80.102
                          ->  Limit  (cost=22301.05..28381.94 rows=51 width=1392) (actual time=175.881..229.983 rows=49 loops=1)
                                Buffers: shared hit=36141, temp read=1388 written=4110
                                I/O Timings: temp read=6.920 write=80.102
                                ->  Result  (cost=22301.05..63913.44 rows=349 width=1392) (actual time=175.879..229.953 rows=49 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=36141, temp read=1388 written=4110
                                      I/O Timings: temp read=6.920 write=80.102
                                      ->  Nested Loop Semi Join  (cost=22301.05..63913.44 rows=349 width=1384) (actual time=175.860..229.876 rows=49 loops=1)
                                            Join Filter: (r.id = v.record_id)
                                            Buffers: shared hit=36141, temp read=1388 written=4110
                                            I/O Timings: temp read=6.920 write=80.102
                                            ->  Nested Loop Semi Join  (cost=22300.61..62874.47 rows=995 width=1400) (actual time=174.703..225.423 rows=870 loops=1)
                                                  Buffers: shared hit=32661, temp read=1388 written=4110
                                                  I/O Timings: temp read=6.920 write=80.102
                                                  ->  Gather Merge  (cost=22300.18..23016.57 rows=6151 width=1384) (actual time=174.468..202.610 rows=2708 loops=1)
                                                        Workers Planned: 2
                                                        Workers Launched: 2
                                                        Buffers: shared hit=21829, temp read=1388 written=4110
                                                        I/O Timings: temp read=6.920 write=80.102
                                                        ->  Sort  (cost=21300.16..21306.57 rows=2563 width=1384) (actual time=162.629..163.591 rows=937 loops=3)
                                                              Sort Key: r.updated_at DESC, r.id DESC
                                                              Sort Method: external merge  Disk: 15072kB
                                                              Buffers: shared hit=21829, temp read=1388 written=4110
                                                              I/O Timings: temp read=6.920 write=80.102
                                                              Worker 0:  Sort Method: external merge  Disk: 8432kB
                                                              Worker 1:  Sort Method: external merge  Disk: 9320kB
                                                              ->  Parallel Bitmap Heap Scan on records r  (cost=480.02..21155.05 rows=2563 width=1384) (actual time=17.807..96.943 rows=8039 loops=3)
                                                                    Recheck Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Heap Blocks: exact=9836
                                                                    Buffers: shared hit=21797
                                                                    ->  Bitmap Index Scan on records_list_group  (cost=0.00..478.48 rows=6151 width=0) (actual time=14.133..14.134 rows=24118 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=375
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..6.48 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=2708)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=10832
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..1.03 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=870)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_1.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=3480
                          ->  Limit  (cost=1.29..1464.83 rows=7 width=1392) (actual time=0.403..1.628 rows=2 loops=1)
                                Buffers: shared hit=763
                                ->  Result  (cost=1.29..1464.83 rows=7 width=1392) (actual time=0.400..1.623 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=763
                                      ->  Nested Loop Semi Join  (cost=1.29..1464.83 rows=7 width=1384) (actual time=0.382..1.602 rows=2 loops=1)
                                            Join Filter: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=763
                                            ->  Nested Loop Semi Join  (cost=0.85..1442.89 rows=21 width=1400) (actual time=0.056..1.382 rows=42 loops=1)
                                                  Buffers: shared hit=595
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.039..0.371 rows=117 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 5
                                                        Buffers: shared hit=127
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=117)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=468
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..1.03 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=42)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_3.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=168
                          ->  Limit  (cost=1.29..148.87 rows=1 width=1392) (actual time=0.178..1.311 rows=1 loops=1)
                                Buffers: shared hit=379
                                ->  Result  (cost=1.29..148.87 rows=1 width=1392) (actual time=0.177..1.307 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=379
                                      ->  Nested Loop Semi Join  (cost=1.29..148.87 rows=1 width=1384) (actual time=0.169..1.296 rows=1 loops=1)
                                            Join Filter: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=379
                                            ->  Nested Loop Semi Join  (cost=0.85..146.77 rows=2 width=1400) (actual time=0.129..1.163 rows=17 loops=1)
                                                  Buffers: shared hit=311
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.025..0.338 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..8.45 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=61)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=244
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..1.03 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=17)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=68
                          ->  Limit  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.262..0.274 rows=0 loops=1)
                                Buffers: shared hit=49
                                ->  Sort  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.261..0.272 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=49
                                      ->  Result  (cost=37.27..46.51 rows=1 width=1392) (actual time=0.244..0.255 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=1384) (actual time=0.238..0.247 rows=0 loops=1)
                                                  Buffers: shared hit=49
                                                  ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=1416) (actual time=0.134..0.199 rows=3 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.108..0.146 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.076..0.082 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.054..0.066 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.040..0.041 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.015 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.017..0.017 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.014..0.015 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                              Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.014..0.014 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.005..0.005 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 8.733 ms
Execution Time: 240.839 ms
```

### ts range 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=12414.11..12414.12 rows=1 width=8) (actual time=217.238..229.926 rows=1 loops=1)
  Buffers: shared hit=190032
  ->  Limit  (cost=10163.13..12401.59 rows=1001 width=4) (actual time=70.610..229.358 rows=1001 loops=1)
        Buffers: shared hit=190032
        ->  Gather  (cost=10163.13..35924.49 rows=11520 width=4) (actual time=70.607..229.038 rows=1001 loops=1)
              Workers Planned: 2
              Workers Launched: 2
              Buffers: shared hit=190032
              ->  Result  (cost=9163.13..33772.49 rows=4800 width=4) (actual time=64.785..210.299 rows=662 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=190032
                    ->  Nested Loop Semi Join  (cost=9163.13..33772.49 rows=4800 width=0) (actual time=64.770..209.939 rows=662 loops=3)
                          Buffers: shared hit=190032
                          ->  Parallel Hash Semi Join  (cost=9162.69..22404.13 rows=13692 width=48) (actual time=64.396..118.568 rows=10466 loops=3)
                                Hash Cond: (r.id = v_1.record_id)
                                Buffers: shared hit=64438
                                ->  Parallel Index Only Scan using records_list_updated on records r  (cost=0.42..12867.28 rows=84672 width=32) (actual time=0.153..22.303 rows=32428 loops=3)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Heap Fetches: 0
                                      Buffers: shared hit=62812
                                ->  Parallel Hash  (cost=8488.49..8488.49 rows=53902 width=32) (actual time=63.482..63.483 rows=42852 loops=3)
                                      Buckets: 131072  Batches: 1  Memory Usage: 0kB
                                      Buffers: shared hit=1568
                                      ->  Parallel Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..8488.49 rows=53902 width=32) (actual time=0.101..31.556 rows=42852 loops=3)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                            Heap Fetches: 0
                                            Buffers: shared hit=1568
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=31398)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=125594
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.431 ms
Execution Time: 230.048 ms
```

### ts range 2%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11
```

```
Limit  (cost=1.85..2265.51 rows=51 width=1426) (actual time=0.197..48.558 rows=51 loops=1)
  Buffers: shared hit=17450
  ->  Result  (cost=1.85..511323.66 rows=11520 width=1426) (actual time=0.194..48.505 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=17450
        ->  Nested Loop Left Join  (cost=1.85..511237.26 rows=11520 width=1399) (actual time=0.170..48.218 rows=51 loops=1)
              Buffers: shared hit=17450
              ->  Nested Loop  (cost=1.55..507468.12 rows=11520 width=1391) (actual time=0.148..46.775 rows=51 loops=1)
                    Buffers: shared hit=17297
                    ->  Nested Loop Semi Join  (cost=1.27..507315.82 rows=11520 width=1384) (actual time=0.128..46.489 rows=51 loops=1)
                          Buffers: shared hit=17294
                          ->  Nested Loop Semi Join  (cost=0.85..480245.03 rows=32861 width=1400) (actual time=0.094..40.260 rows=869 loops=1)
                                Buffers: shared hit=13818
                                ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.045..9.073 rows=2757 loops=1)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Buffers: shared hit=2790
                                ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.82 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=2757)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                      Filter: (value < '2024-08-10 05:20:00+00'::timestamp with time zone)
                                      Rows Removed by Filter: 1
                                      Buffers: shared hit=11028
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=869)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=3476
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.013..0.014 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.025..0.025 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.014..0.014 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.261 ms
Execution Time: 48.769 ms
```

### ts range 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=15298.94..15298.95 rows=1 width=8) (actual time=270.726..270.746 rows=1 loops=1)
  Buffers: shared hit=50417, temp read=464 written=464
  I/O Timings: temp read=3.575 write=15.330
  ->  Limit  (cost=11122.32..15294.70 rows=339 width=4) (actual time=114.847..270.572 rows=69 loops=1)
        Buffers: shared hit=50417, temp read=464 written=464
        I/O Timings: temp read=3.575 write=15.330
        ->  Result  (cost=11122.32..15294.70 rows=339 width=4) (actual time=114.845..270.503 rows=69 loops=1)
              Buffers: shared hit=50417, temp read=464 written=464
              I/O Timings: temp read=3.575 write=15.330
              ->  Append  (cost=11122.32..15291.31 rows=339 width=0) (actual time=114.843..270.425 rows=69 loops=1)
                    Buffers: shared hit=50417, temp read=464 written=464
                    I/O Timings: temp read=3.575 write=15.330
                    ->  Subquery Scan on "*SELECT* 1"  (cost=11122.32..13629.66 rows=330 width=0) (actual time=114.841..261.764 rows=68 loops=1)
                          Buffers: shared hit=46878, temp read=464 written=464
                          I/O Timings: temp read=3.575 write=15.330
                          ->  Limit  (cost=11122.32..13626.36 rows=330 width=4) (actual time=114.840..261.679 rows=68 loops=1)
                                Buffers: shared hit=46878, temp read=464 written=464
                                I/O Timings: temp read=3.575 write=15.330
                                ->  Result  (cost=11122.32..13626.36 rows=330 width=4) (actual time=114.838..261.615 rows=68 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=46878, temp read=464 written=464
                                      I/O Timings: temp read=3.575 write=15.330
                                      ->  Nested Loop Semi Join  (cost=11122.32..13626.36 rows=330 width=0) (actual time=114.826..261.405 rows=68 loops=1)
                                            Join Filter: (r.id = v.record_id)
                                            Buffers: shared hit=46878, temp read=464 written=464
                                            I/O Timings: temp read=3.575 write=15.330
                                            ->  Hash Semi Join  (cost=11121.88..12611.95 rows=942 width=48) (actual time=110.940..179.026 rows=7429 loops=1)
                                                  Hash Cond: (r.id = v_1.record_id)
                                                  Buffers: shared hit=17162, temp read=464 written=464
                                                  I/O Timings: temp read=3.575 write=15.330
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.036..16.447 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                                  ->  Hash  (cost=8752.29..8752.29 rows=122483 width=32) (actual time=110.771..110.772 rows=121439 loops=1)
                                                        Buckets: 131072  Batches: 2  Memory Usage: 4807kB
                                                        Buffers: shared hit=1481, temp written=386
                                                        I/O Timings: temp write=12.040
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..8752.29 rows=122483 width=32) (actual time=0.028..44.562 rows=121439 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=1481
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..1.06 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=7429)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_1.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=29716
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.29..1464.50 rows=7 width=0) (actual time=5.469..7.637 rows=1 loops=1)
                          Buffers: shared hit=3115
                          ->  Limit  (cost=1.29..1464.43 rows=7 width=4) (actual time=5.468..7.632 rows=1 loops=1)
                                Buffers: shared hit=3115
                                ->  Result  (cost=1.29..1464.43 rows=7 width=4) (actual time=5.463..7.626 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=3115
                                      ->  Nested Loop Semi Join  (cost=1.29..1464.43 rows=7 width=0) (actual time=5.440..7.601 rows=1 loops=1)
                                            Join Filter: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=3115
                                            ->  Nested Loop Semi Join  (cost=0.85..1442.88 rows=20 width=48) (actual time=0.109..6.527 rows=164 loops=1)
                                                  Buffers: shared hit=2459
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.078..1.569 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=484)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=1936
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..1.06 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=164)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_3.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=656
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.29..148.94 rows=1 width=0) (actual time=0.769..0.771 rows=0 loops=1)
                          Buffers: shared hit=375
                          ->  Limit  (cost=1.29..148.93 rows=1 width=4) (actual time=0.768..0.770 rows=0 loops=1)
                                Buffers: shared hit=375
                                ->  Result  (cost=1.29..148.93 rows=1 width=4) (actual time=0.763..0.765 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=375
                                      ->  Nested Loop Semi Join  (cost=1.29..148.93 rows=1 width=0) (actual time=0.747..0.748 rows=0 loops=1)
                                            Join Filter: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=375
                                            ->  Nested Loop Semi Join  (cost=0.85..146.77 rows=2 width=48) (actual time=0.164..0.669 rows=16 loops=1)
                                                  Buffers: shared hit=311
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.052..0.164 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=61)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=244
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..1.06 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=16)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=64
                    ->  Subquery Scan on "*SELECT* 4"  (cost=37.27..46.52 rows=1 width=0) (actual time=0.164..0.167 rows=0 loops=1)
                          Buffers: shared hit=49
                          ->  Limit  (cost=37.27..46.51 rows=1 width=4) (actual time=0.162..0.166 rows=0 loops=1)
                                Buffers: shared hit=49
                                ->  Result  (cost=37.27..46.51 rows=1 width=4) (actual time=0.161..0.164 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=49
                                      ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=0) (actual time=0.154..0.157 rows=0 loops=1)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=64) (actual time=0.100..0.135 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.084..0.104 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.058 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.045 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.016 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.014..0.014 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.431 ms
Execution Time: 271.005 ms
```

### ts range 0.25%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $12
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22350.76..29200.97 rows=1 width=1434) (actual time=146.226..344.181 rows=51 loops=1)
  Buffers: shared hit=110184, temp read=3468 written=4112
  I/O Timings: temp read=11.597 write=55.089
  ->  Nested Loop  (cost=22350.46..29192.64 rows=1 width=1399) (actual time=146.199..343.227 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=110031, temp read=3468 written=4112
        I/O Timings: temp read=11.597 write=55.089
        ->  Subquery Scan on x  (cost=22350.18..29184.32 rows=1 width=1392) (actual time=146.177..342.524 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=109878, temp read=3468 written=4112
              I/O Timings: temp read=11.597 write=55.089
              ->  Limit  (cost=22350.18..29183.18 rows=51 width=1392) (actual time=146.172..342.047 rows=51 loops=1)
                    Buffers: shared hit=109878, temp read=3468 written=4112
                    I/O Timings: temp read=11.597 write=55.089
                    ->  Merge Append  (cost=22350.18..30389.00 rows=60 width=1392) (actual time=146.170..341.982 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=109878, temp read=3468 written=4112
                          I/O Timings: temp read=11.597 write=55.089
                          ->  Limit  (cost=22301.05..28728.17 rows=51 width=1392) (actual time=141.395..337.106 rows=51 loops=1)
                                Buffers: shared hit=107177, temp read=3468 written=4112
                                I/O Timings: temp read=11.597 write=55.089
                                ->  Result  (cost=22301.05..63888.35 rows=330 width=1392) (actual time=141.392..337.055 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=107177, temp read=3468 written=4112
                                      I/O Timings: temp read=11.597 write=55.089
                                      ->  Nested Loop Semi Join  (cost=22301.05..63888.35 rows=330 width=1384) (actual time=141.384..336.932 rows=51 loops=1)
                                            Join Filter: (r.id = v.record_id)
                                            Buffers: shared hit=107177, temp read=3468 written=4112
                                            I/O Timings: temp read=11.597 write=55.089
                                            ->  Nested Loop Semi Join  (cost=22300.61..62873.94 rows=942 width=1400) (actual time=130.152..313.595 rows=4984 loops=1)
                                                  Buffers: shared hit=87241, temp read=3468 written=4112
                                                  I/O Timings: temp read=11.597 write=55.089
                                                  ->  Gather Merge  (cost=22300.18..23016.57 rows=6151 width=1384) (actual time=130.021..189.526 rows=16353 loops=1)
                                                        Workers Planned: 2
                                                        Workers Launched: 2
                                                        Buffers: shared hit=21829, temp read=3468 written=4112
                                                        I/O Timings: temp read=11.597 write=55.089
                                                        ->  Sort  (cost=21300.16..21306.57 rows=2563 width=1384) (actual time=121.318..129.414 rows=5477 loops=3)
                                                              Sort Key: r.updated_at DESC, r.id DESC
                                                              Sort Method: external merge  Disk: 15608kB
                                                              Buffers: shared hit=21829, temp read=3468 written=4112
                                                              I/O Timings: temp read=11.597 write=55.089
                                                              Worker 0:  Sort Method: external merge  Disk: 8968kB
                                                              Worker 1:  Sort Method: external merge  Disk: 8264kB
                                                              ->  Parallel Bitmap Heap Scan on records r  (cost=480.02..21155.05 rows=2563 width=1384) (actual time=12.084..72.216 rows=8039 loops=3)
                                                                    Recheck Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Heap Blocks: exact=10212
                                                                    Buffers: shared hit=21797
                                                                    ->  Bitmap Index Scan on records_list_group  (cost=0.00..478.48 rows=6151 width=0) (actual time=10.255..10.255 rows=24118 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=375
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..6.48 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=16353)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=65412
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..1.06 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=4984)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_1.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=19936
                          ->  Limit  (cost=1.29..1464.43 rows=7 width=1392) (actual time=3.915..3.921 rows=1 loops=1)
                                Buffers: shared hit=2277
                                ->  Result  (cost=1.29..1464.43 rows=7 width=1392) (actual time=3.913..3.918 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2277
                                      ->  Nested Loop Semi Join  (cost=1.29..1464.43 rows=7 width=1384) (actual time=3.896..3.900 rows=1 loops=1)
                                            Join Filter: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=2277
                                            ->  Nested Loop Semi Join  (cost=0.85..1442.88 rows=20 width=1400) (actual time=0.082..3.403 rows=126 loops=1)
                                                  Buffers: shared hit=1773
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.064..0.871 rows=349 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 19
                                                        Buffers: shared hit=377
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..8.45 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=349)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=1396
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..1.06 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=126)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_3.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=504
                          ->  Limit  (cost=1.29..148.93 rows=1 width=1392) (actual time=0.669..0.675 rows=0 loops=1)
                                Buffers: shared hit=375
                                ->  Result  (cost=1.29..148.93 rows=1 width=1392) (actual time=0.667..0.671 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=375
                                      ->  Nested Loop Semi Join  (cost=1.29..148.93 rows=1 width=1384) (actual time=0.654..0.658 rows=0 loops=1)
                                            Join Filter: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=375
                                            ->  Nested Loop Semi Join  (cost=0.85..146.77 rows=2 width=1400) (actual time=0.131..0.595 rows=16 loops=1)
                                                  Buffers: shared hit=311
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.029..0.154 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..8.45 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=61)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=244
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..1.06 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=16)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=64
                          ->  Limit  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.184..0.195 rows=0 loops=1)
                                Buffers: shared hit=49
                                ->  Sort  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.183..0.193 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=49
                                      ->  Result  (cost=37.27..46.51 rows=1 width=1392) (actual time=0.172..0.182 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=1384) (actual time=0.168..0.177 rows=0 loops=1)
                                                  Buffers: shared hit=49
                                                  ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=1416) (actual time=0.113..0.154 rows=3 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.083..0.108 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.057..0.062 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.041..0.050 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.032 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.012..0.012 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.017 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.013..0.013 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.013..0.013 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                              Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.008..0.008 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 5.800 ms
Execution Time: 348.912 ms
```

### ts range 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=11991.55..11991.56 rows=1 width=8) (actual time=787.067..808.474 rows=1 loops=1)
  Buffers: shared hit=745459
  ->  Limit  (cost=9676.61..11979.04 rows=1001 width=4) (actual time=47.671..807.468 rows=1001 loops=1)
        Buffers: shared hit=745459
        ->  Gather  (cost=9676.61..34764.12 rows=10907 width=4) (actual time=47.668..806.723 rows=1001 loops=1)
              Workers Planned: 2
              Workers Launched: 2
              Buffers: shared hit=745459
              ->  Result  (cost=8676.61..32673.42 rows=4545 width=4) (actual time=47.996..778.557 rows=347 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=745459
                    ->  Nested Loop Semi Join  (cost=8676.61..32673.42 rows=4545 width=0) (actual time=47.982..778.083 rows=347 loops=3)
                          Buffers: shared hit=745459
                          ->  Parallel Hash Semi Join  (cost=8676.17..21909.51 rows=12964 width=48) (actual time=40.215..328.264 rows=40480 loops=3)
                                Hash Cond: (r.id = v_1.record_id)
                                Buffers: shared hit=259701
                                ->  Parallel Index Only Scan using records_list_updated on records r  (cost=0.42..12867.28 rows=84672 width=32) (actual time=0.236..123.631 rows=133333 loops=3)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Heap Fetches: 0
                                      Buffers: shared hit=258160
                                ->  Parallel Hash  (cost=8037.81..8037.81 rows=51035 width=32) (actual time=39.128..39.130 rows=40480 loops=3)
                                      Buckets: 131072  Batches: 1  Memory Usage: 8704kB
                                      Buffers: shared hit=1483
                                      ->  Parallel Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..8037.81 rows=51035 width=32) (actual time=0.085..19.953 rows=40480 loops=3)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                            Heap Fetches: 0
                                            Buffers: shared hit=1483
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=121439)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=485758
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.434 ms
Execution Time: 808.600 ms
```

### ts range 0.25%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11
```

```
Limit  (cost=1.85..2384.93 rows=51 width=1426) (actual time=11.304..327.205 rows=51 loops=1)
  Buffers: shared hit=126555
  ->  Result  (cost=1.85..509653.36 rows=10907 width=1426) (actual time=11.301..327.120 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=126555
        ->  Nested Loop Left Join  (cost=1.85..509571.55 rows=10907 width=1399) (actual time=11.272..326.582 rows=51 loops=1)
              Buffers: shared hit=126555
              ->  Nested Loop  (cost=1.55..506002.98 rows=10907 width=1391) (actual time=11.219..324.661 rows=51 loops=1)
                    Buffers: shared hit=126402
                    ->  Nested Loop Semi Join  (cost=1.27..505858.35 rows=10907 width=1384) (actual time=11.185..324.253 rows=51 loops=1)
                          Buffers: shared hit=126399
                          ->  Nested Loop Semi Join  (cost=0.85..480227.55 rows=31113 width=1400) (actual time=0.151..283.390 rows=6180 loops=1)
                                Buffers: shared hit=101679
                                ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.097..80.208 rows=20288 loops=1)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Buffers: shared hit=20527
                                ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.82 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=20288)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                      Filter: (value < '2024-07-29 01:40:00+00'::timestamp with time zone)
                                      Rows Removed by Filter: 1
                                      Buffers: shared hit=81152
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=6180)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=24720
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.002..0.003 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.019..0.020 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.033..0.033 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.976 ms
Execution Time: 327.428 ms
```

### ts range 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=15208.07..15208.08 rows=1 width=8) (actual time=301.045..301.066 rows=1 loops=1)
  Buffers: shared hit=50144, temp read=461 written=461
  I/O Timings: temp read=1.977 write=18.661
  ->  Limit  (cost=11040.78..15203.86 rows=337 width=4) (actual time=196.803..301.043 rows=4 loops=1)
        Buffers: shared hit=50144, temp read=461 written=461
        I/O Timings: temp read=1.977 write=18.661
        ->  Result  (cost=11040.78..15203.86 rows=337 width=4) (actual time=196.800..301.034 rows=4 loops=1)
              Buffers: shared hit=50144, temp read=461 written=461
              I/O Timings: temp read=1.977 write=18.661
              ->  Append  (cost=11040.78..15200.49 rows=337 width=0) (actual time=196.799..301.024 rows=4 loops=1)
                    Buffers: shared hit=50144, temp read=461 written=461
                    I/O Timings: temp read=1.977 write=18.661
                    ->  Subquery Scan on "*SELECT* 1"  (cost=11040.78..13538.74 rows=328 width=0) (actual time=196.796..295.422 rows=4 loops=1)
                          Buffers: shared hit=46609, temp read=461 written=461
                          I/O Timings: temp read=1.977 write=18.661
                          ->  Limit  (cost=11040.78..13535.46 rows=328 width=4) (actual time=196.794..295.411 rows=4 loops=1)
                                Buffers: shared hit=46609, temp read=461 written=461
                                I/O Timings: temp read=1.977 write=18.661
                                ->  Result  (cost=11040.78..13535.46 rows=328 width=4) (actual time=196.792..295.404 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=46609, temp read=461 written=461
                                      I/O Timings: temp read=1.977 write=18.661
                                      ->  Nested Loop Semi Join  (cost=11040.78..13535.46 rows=328 width=0) (actual time=196.774..295.377 rows=4 loops=1)
                                            Join Filter: (r.id = v.record_id)
                                            Buffers: shared hit=46609, temp read=461 written=461
                                            I/O Timings: temp read=1.977 write=18.661
                                            ->  Hash Semi Join  (cost=11040.34..12524.33 rows=935 width=48) (actual time=151.395..218.360 rows=7365 loops=1)
                                                  Hash Cond: (r.id = v_1.record_id)
                                                  Buffers: shared hit=17149, temp read=461 written=461
                                                  I/O Timings: temp read=1.977 write=18.661
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.059..21.172 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                                  ->  Hash  (cost=8688.02..8688.02 rows=121582 width=32) (actual time=151.123..151.124 rows=120443 loops=1)
                                                        Buckets: 131072  Batches: 2  Memory Usage: 4777kB
                                                        Buffers: shared hit=1468, temp written=383
                                                        I/O Timings: temp write=13.628
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..8688.02 rows=121582 width=32) (actual time=0.040..60.115 rows=120443 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=1468
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..1.07 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=7365)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_1.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=29460
                    ->  Subquery Scan on "*SELECT* 2"  (cost=1.29..1464.59 rows=7 width=0) (actual time=4.696..4.700 rows=0 loops=1)
                          Buffers: shared hit=3111
                          ->  Limit  (cost=1.29..1464.52 rows=7 width=4) (actual time=4.694..4.697 rows=0 loops=1)
                                Buffers: shared hit=3111
                                ->  Result  (cost=1.29..1464.52 rows=7 width=4) (actual time=4.690..4.693 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=3111
                                      ->  Nested Loop Semi Join  (cost=1.29..1464.52 rows=7 width=0) (actual time=4.675..4.677 rows=0 loops=1)
                                            Join Filter: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=3111
                                            ->  Nested Loop Semi Join  (cost=0.85..1442.88 rows=20 width=48) (actual time=0.075..4.114 rows=163 loops=1)
                                                  Buffers: shared hit=2459
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.058..1.049 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=484)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=1936
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..1.07 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=163)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_3.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=652
                    ->  Subquery Scan on "*SELECT* 3"  (cost=1.29..148.95 rows=1 width=0) (actual time=0.708..0.711 rows=0 loops=1)
                          Buffers: shared hit=375
                          ->  Limit  (cost=1.29..148.94 rows=1 width=4) (actual time=0.707..0.710 rows=0 loops=1)
                                Buffers: shared hit=375
                                ->  Result  (cost=1.29..148.94 rows=1 width=4) (actual time=0.704..0.706 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=375
                                      ->  Nested Loop Semi Join  (cost=1.29..148.94 rows=1 width=0) (actual time=0.690..0.692 rows=0 loops=1)
                                            Join Filter: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=375
                                            ->  Nested Loop Semi Join  (cost=0.85..146.77 rows=2 width=48) (actual time=0.133..0.617 rows=16 loops=1)
                                                  Buffers: shared hit=311
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.046..0.162 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..8.45 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=61)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=244
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..1.07 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=16)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=64
                    ->  Subquery Scan on "*SELECT* 4"  (cost=37.27..46.52 rows=1 width=0) (actual time=0.171..0.175 rows=0 loops=1)
                          Buffers: shared hit=49
                          ->  Limit  (cost=37.27..46.51 rows=1 width=4) (actual time=0.170..0.173 rows=0 loops=1)
                                Buffers: shared hit=49
                                ->  Result  (cost=37.27..46.51 rows=1 width=4) (actual time=0.167..0.170 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=49
                                      ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=0) (actual time=0.160..0.163 rows=0 loops=1)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=64) (actual time=0.110..0.142 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.095..0.114 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.059..0.062 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.048 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.030..0.031 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.014 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.015..0.015 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.106 ms
Execution Time: 301.477 ms
```

### ts range 0.01%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $10::uuid AND v.value < $11::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $12)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $12
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22350.76..29233.92 rows=1 width=1434) (actual time=249.125..664.653 rows=4 loops=1)
  Buffers: shared hit=151320, temp read=4102 written=4108
  I/O Timings: temp read=16.701 write=59.376
  ->  Nested Loop  (cost=22350.46..29225.59 rows=1 width=1399) (actual time=249.088..664.534 rows=4 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=151308, temp read=4102 written=4108
        I/O Timings: temp read=16.701 write=59.376
        ->  Subquery Scan on x  (cost=22350.18..29217.28 rows=1 width=1392) (actual time=249.061..664.445 rows=4 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=151296, temp read=4102 written=4108
              I/O Timings: temp read=16.701 write=59.376
              ->  Limit  (cost=22350.18..29216.13 rows=51 width=1392) (actual time=249.053..664.346 rows=4 loops=1)
                    Buffers: shared hit=151296, temp read=4102 written=4108
                    I/O Timings: temp read=16.701 write=59.376
                    ->  Merge Append  (cost=22350.18..30427.77 rows=60 width=1392) (actual time=249.050..664.335 rows=4 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=151296, temp read=4102 written=4108
                          I/O Timings: temp read=16.701 write=59.376
                          ->  Limit  (cost=22301.05..28766.84 rows=51 width=1392) (actual time=240.831..656.082 rows=4 loops=1)
                                Buffers: shared hit=147761, temp read=4102 written=4108
                                I/O Timings: temp read=16.701 write=59.376
                                ->  Result  (cost=22301.05..63885.00 rows=328 width=1392) (actual time=240.829..656.074 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=147761, temp read=4102 written=4108
                                      I/O Timings: temp read=16.701 write=59.376
                                      ->  Nested Loop Semi Join  (cost=22301.05..63885.00 rows=328 width=1384) (actual time=240.815..656.045 rows=4 loops=1)
                                            Join Filter: (r.id = v.record_id)
                                            Buffers: shared hit=147761, temp read=4102 written=4108
                                            I/O Timings: temp read=16.701 write=59.376
                                            ->  Nested Loop Semi Join  (cost=22300.61..62873.87 rows=935 width=1400) (actual time=165.365..603.984 rows=7365 loops=1)
                                                  Buffers: shared hit=118301, temp read=4102 written=4108
                                                  I/O Timings: temp read=16.701 write=59.376
                                                  ->  Gather Merge  (cost=22300.18..23016.57 rows=6151 width=1384) (actual time=165.061..298.544 rows=24118 loops=1)
                                                        Workers Planned: 2
                                                        Workers Launched: 2
                                                        Buffers: shared hit=21829, temp read=4102 written=4108
                                                        I/O Timings: temp read=16.701 write=59.376
                                                        ->  Sort  (cost=21300.16..21306.57 rows=2563 width=1384) (actual time=152.873..168.609 rows=8039 loops=3)
                                                              Sort Key: r.updated_at DESC, r.id DESC
                                                              Sort Method: external merge  Disk: 14968kB
                                                              Buffers: shared hit=21829, temp read=4102 written=4108
                                                              I/O Timings: temp read=16.701 write=59.376
                                                              Worker 0:  Sort Method: external merge  Disk: 6336kB
                                                              Worker 1:  Sort Method: external merge  Disk: 11512kB
                                                              ->  Parallel Bitmap Heap Scan on records r  (cost=480.02..21155.05 rows=2563 width=1384) (actual time=15.167..99.936 rows=8039 loops=3)
                                                                    Recheck Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                    Heap Blocks: exact=9750
                                                                    Buffers: shared hit=21797
                                                                    ->  Bitmap Index Scan on records_list_group  (cost=0.00..478.48 rows=6151 width=0) (actual time=10.744..10.744 rows=24118 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=375
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..6.48 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=24118)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=96472
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..1.07 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=7365)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_1.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=29460
                          ->  Limit  (cost=1.29..1464.52 rows=7 width=1392) (actual time=6.901..6.906 rows=0 loops=1)
                                Buffers: shared hit=3111
                                ->  Result  (cost=1.29..1464.52 rows=7 width=1392) (actual time=6.897..6.901 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=3111
                                      ->  Nested Loop Semi Join  (cost=1.29..1464.52 rows=7 width=1384) (actual time=6.881..6.884 rows=0 loops=1)
                                            Join Filter: (r_1.id = v_2.record_id)
                                            Buffers: shared hit=3111
                                            ->  Nested Loop Semi Join  (cost=0.85..1442.88 rows=20 width=1400) (actual time=0.070..6.074 rows=163 loops=1)
                                                  Buffers: shared hit=2459
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.033..1.482 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=484)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_1.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=1936
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..1.07 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=163)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_3.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=652
                          ->  Limit  (cost=1.29..148.94 rows=1 width=1392) (actual time=0.992..0.998 rows=0 loops=1)
                                Buffers: shared hit=375
                                ->  Result  (cost=1.29..148.94 rows=1 width=1392) (actual time=0.988..0.992 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=375
                                      ->  Nested Loop Semi Join  (cost=1.29..148.94 rows=1 width=1384) (actual time=0.970..0.973 rows=0 loops=1)
                                            Join Filter: (r_2.id = v_4.record_id)
                                            Buffers: shared hit=375
                                            ->  Nested Loop Semi Join  (cost=0.85..146.77 rows=2 width=1400) (actual time=0.173..0.875 rows=16 loops=1)
                                                  Buffers: shared hit=311
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.041..0.216 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_5  (cost=0.42..8.45 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=61)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_2.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=244
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_4  (cost=0.42..1.07 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=16)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_5.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                  Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=64
                          ->  Limit  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.318..0.329 rows=0 loops=1)
                                Buffers: shared hit=49
                                ->  Sort  (cost=46.52..46.52 rows=1 width=1392) (actual time=0.316..0.326 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=49
                                      ->  Result  (cost=37.27..46.51 rows=1 width=1392) (actual time=0.276..0.285 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=49
                                            ->  Nested Loop Semi Join  (cost=37.27..46.51 rows=1 width=1384) (actual time=0.262..0.270 rows=0 loops=1)
                                                  Buffers: shared hit=49
                                                  ->  Nested Loop Semi Join  (cost=36.83..45.77 rows=1 width=1416) (actual time=0.183..0.240 rows=3 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.143..0.177 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.089..0.095 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.061..0.073 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.045..0.047 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.024..0.024 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_ts_pkey on rfv_ts v_6  (cost=0.42..0.87 rows=1 width=32) (actual time=0.018..0.018 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r_3.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                              Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_7  (cost=0.42..0.72 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = v_6.record_id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                                        Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.013..0.013 rows=1 loops=4)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=12
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.013..0.013 rows=1 loops=4)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.092 ms
Execution Time: 671.150 ms
```

### ts range 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=11936.64..11936.65 rows=1 width=8) (actual time=379.433..397.246 rows=1 loops=1)
  Buffers: shared hit=741478
  ->  Limit  (cost=9612.89..11924.13 rows=1001 width=4) (actual time=54.372..397.197 rows=46 loops=1)
        Buffers: shared hit=741478
        ->  Gather  (cost=9612.89..34611.63 rows=10827 width=4) (actual time=54.370..397.171 rows=46 loops=1)
              Workers Planned: 2
              Workers Launched: 2
              Buffers: shared hit=741478
              ->  Result  (cost=8612.89..32528.93 rows=4511 width=4) (actual time=59.962..372.750 rows=15 loops=3)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=741478
                    ->  Nested Loop Semi Join  (cost=8612.89..32528.93 rows=4511 width=0) (actual time=59.952..372.723 rows=15 loops=3)
                          Buffers: shared hit=741478
                          ->  Parallel Hash Semi Join  (cost=8612.45..21844.73 rows=12868 width=48) (actual time=36.091..173.722 rows=40148 loops=3)
                                Hash Cond: (r.id = v_1.record_id)
                                Buffers: shared hit=259704
                                ->  Parallel Index Only Scan using records_list_updated on records r  (cost=0.42..12867.28 rows=84672 width=32) (actual time=0.100..57.941 rows=133333 loops=3)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Heap Fetches: 0
                                      Buffers: shared hit=258176
                                ->  Parallel Hash  (cost=7978.79..7978.79 rows=50659 width=32) (actual time=35.208..35.210 rows=40148 loops=3)
                                      Buckets: 131072  Batches: 1  Memory Usage: 8608kB
                                      Buffers: shared hit=1470
                                      ->  Parallel Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..7978.79 rows=50659 width=32) (actual time=0.090..17.519 rows=40148 loops=3)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                            Heap Fetches: 0
                                            Buffers: shared hit=1470
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=120443)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=481774
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.316 ms
Execution Time: 397.351 ms
```

### ts range 0.01%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $9::uuid AND v.value < $10::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11
```

```
Limit  (cost=1.85..2401.50 rows=51 width=1426) (actual time=158.076..2848.872 rows=46 loops=1)
  Buffers: shared hit=2486728
  ->  Result  (cost=1.85..509434.64 rows=10827 width=1426) (actual time=158.073..2848.806 rows=46 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=2486728
        ->  Nested Loop Left Join  (cost=1.85..509353.44 rows=10827 width=1399) (actual time=158.050..2848.510 rows=46 loops=1)
              Buffers: shared hit=2486728
              ->  Nested Loop  (cost=1.55..505811.04 rows=10827 width=1391) (actual time=158.015..2847.337 rows=46 loops=1)
                    Buffers: shared hit=2486590
                    ->  Nested Loop Semi Join  (cost=1.27..505667.41 rows=10827 width=1384) (actual time=157.984..2847.092 rows=46 loops=1)
                          Buffers: shared hit=2486587
                          ->  Nested Loop Semi Join  (cost=0.85..480225.26 rows=30884 width=1400) (actual time=0.099..2515.464 rows=120443 loops=1)
                                Buffers: shared hit=2004815
                                ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.037..780.317 rows=400000 loops=1)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                      Buffers: shared hit=404815
                                ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..0.82 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=400000)
                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                      Filter: (value < '2024-07-27 09:40:00+00'::timestamp with time zone)
                                      Rows Removed by Filter: 1
                                      Buffers: shared hit=1600000
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.82 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=120443)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (record_id = r.id) AND (field_id = 'e54ba9a9-45d0-4dce-a54f-fcf22d96d35b'::uuid))
                                Filter: (value >= '2024-07-27 08:00:00+00'::timestamp with time zone)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=481772
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.001..0.002 rows=1 loops=46)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.017..0.018 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.022..0.022 rows=1 loops=46)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=138
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=46)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=138
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.837 ms
Execution Time: 2849.063 ms
```

### bool eq 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=11351.72..11351.73 rows=1 width=8) (actual time=31.805..31.818 rows=1 loops=1)
  Buffers: shared hit=15885
  ->  Limit  (cost=987.03..11339.20 rows=1001 width=4) (actual time=23.530..31.665 rows=1001 loops=1)
        Buffers: shared hit=15885
        ->  Result  (cost=987.03..11649.46 rows=1031 width=4) (actual time=23.526..31.498 rows=1001 loops=1)
              Buffers: shared hit=15885
              ->  Append  (cost=987.03..11639.15 rows=1031 width=0) (actual time=23.525..31.314 rows=1001 loops=1)
                    Buffers: shared hit=15885
                    ->  Subquery Scan on "*SELECT* 1"  (cost=987.03..10554.16 rows=1001 width=0) (actual time=23.523..31.164 rows=1001 loops=1)
                          Buffers: shared hit=15885
                          ->  Limit  (cost=987.03..10544.15 rows=1001 width=4) (actual time=23.521..30.999 rows=1001 loops=1)
                                Buffers: shared hit=15885
                                ->  Result  (cost=987.03..12787.82 rows=1236 width=4) (actual time=23.520..30.865 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15885
                                      ->  Merge Semi Join  (cost=987.03..12787.82 rows=1236 width=0) (actual time=23.505..30.645 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15885
                                            ->  Sort  (cost=927.09..942.47 rows=6151 width=32) (actual time=23.421..24.127 rows=2474 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: quicksort  Memory: 2087kB
                                                  Buffers: shared hit=15681
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.057..11.366 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..11474.94 rows=160817 width=32) (actual time=0.069..3.866 rows=17463 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=204
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..937.29 rows=26 width=0) (never executed)
                          ->  Limit  (cost=0.99..937.03 rows=26 width=4) (never executed)
                                ->  Result  (cost=0.99..937.03 rows=26 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..937.03 rows=26 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..96.61 rows=3 width=0) (never executed)
                          ->  Limit  (cost=0.99..96.58 rows=3 width=4) (never executed)
                                ->  Result  (cost=0.99..96.58 rows=3 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..96.58 rows=3 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.94 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.98..45.93 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.98..45.93 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 8.424 ms
Execution Time: 32.316 ms
```

### bool eq 40%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=49.53..9567.85 rows=1 width=1434) (actual time=0.677..6.863 rows=51 loops=1)
  Buffers: shared hit=2665
  ->  Nested Loop  (cost=49.24..9559.51 rows=1 width=1399) (actual time=0.647..6.488 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2512
        ->  Subquery Scan on x  (cost=48.95..9551.20 rows=1 width=1392) (actual time=0.624..6.204 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2359
              ->  Limit  (cost=48.95..9550.05 rows=51 width=1392) (actual time=0.619..6.081 rows=51 loops=1)
                    Buffers: shared hit=2359
                    ->  Merge Append  (cost=48.95..15138.93 rows=81 width=1392) (actual time=0.616..6.062 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2359
                          ->  Limit  (cost=0.99..14058.13 rows=51 width=1392) (actual time=0.194..5.546 rows=50 loops=1)
                                Buffers: shared hit=2278
                                ->  Result  (cost=0.99..340679.74 rows=1236 width=1392) (actual time=0.193..5.530 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2278
                                      ->  Nested Loop Semi Join  (cost=0.99..340679.74 rows=1236 width=1384) (actual time=0.178..5.488 rows=50 loops=1)
                                            Buffers: shared hit=2278
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..314238.64 rows=6151 width=1384) (actual time=0.132..3.974 rows=117 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1670
                                                  Buffers: shared hit=1809
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..4.29 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=117)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=469
                          ->  Limit  (cost=0.99..937.03 rows=26 width=1392) (actual time=0.100..0.153 rows=2 loops=1)
                                Buffers: shared hit=24
                                ->  Result  (cost=0.99..937.03 rows=26 width=1392) (actual time=0.099..0.151 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=24
                                      ->  Nested Loop Semi Join  (cost=0.99..937.03 rows=26 width=1384) (actual time=0.093..0.142 rows=2 loops=1)
                                            Buffers: shared hit=24
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.037..0.048 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=7
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (actual time=0.021..0.021 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=17
                          ->  Limit  (cost=0.99..96.58 rows=3 width=1392) (actual time=0.093..0.097 rows=1 loops=1)
                                Buffers: shared hit=19
                                ->  Result  (cost=0.99..96.58 rows=3 width=1392) (actual time=0.093..0.095 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=19
                                      ->  Nested Loop Semi Join  (cost=0.99..96.58 rows=3 width=1384) (actual time=0.088..0.090 rows=1 loops=1)
                                            Buffers: shared hit=19
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.033..0.041 rows=3 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=6
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.014..0.014 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13
                          ->  Limit  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.223..0.232 rows=1 loops=1)
                                Buffers: shared hit=38
                                ->  Sort  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.222..0.231 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=38
                                      ->  Result  (cost=36.98..45.93 rows=1 width=1392) (actual time=0.148..0.215 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=38
                                            ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=1384) (actual time=0.145..0.209 rows=1 loops=1)
                                                  Buffers: shared hit=38
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.112..0.147 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.078..0.083 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.055..0.068 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.042..0.044 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.021..0.021 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.018..0.019 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.018..0.018 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 21.019 ms
Execution Time: 7.503 ms
```

### bool eq 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=7126.10..7126.11 rows=1 width=8) (actual time=16.566..16.569 rows=1 loops=1)
  Buffers: shared hit=11763
  ->  Limit  (cost=0.99..7113.59 rows=1001 width=4) (actual time=0.151..16.383 rows=1001 loops=1)
        Buffers: shared hit=11763
        ->  Result  (cost=0.99..290260.37 rows=40850 width=4) (actual time=0.149..16.220 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=11763
              ->  Nested Loop Semi Join  (cost=0.99..290260.37 rows=40850 width=0) (actual time=0.138..16.005 rows=1001 loops=1)
                    Buffers: shared hit=11763
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.038..1.136 rows=2529 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1646
                    ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..1.35 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2529)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r.id))
                          Heap Fetches: 0
                          Buffers: shared hit=10117
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.051 ms
Execution Time: 16.626 ms
```

### bool eq 40%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1.55..753.13 rows=51 width=1426) (actual time=0.200..1.788 rows=51 loops=1)
  Buffers: shared hit=816
  ->  Result  (cost=1.55..601997.33 rows=40850 width=1426) (actual time=0.198..1.774 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
        Buffers: shared hit=816
        ->  Nested Loop Left Join  (cost=1.55..601690.95 rows=40850 width=1399) (actual time=0.180..1.711 rows=51 loops=1)
              Buffers: shared hit=816
              ->  Nested Loop  (cost=1.25..588325.53 rows=40850 width=1391) (actual time=0.163..1.410 rows=51 loops=1)
                    Buffers: shared hit=663
                    ->  Nested Loop Semi Join  (cost=0.98..587806.61 rows=40850 width=1384) (actual time=0.147..1.351 rows=51 loops=1)
                          Buffers: shared hit=660
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..313222.57 rows=203213 width=1384) (actual time=0.035..0.317 rows=131 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                                Buffers: shared hit=135
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..1.35 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=131)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '44471a9d-43db-4247-882c-d629127a6204'::uuid) AND (value = true) AND (record_id = r.id))
                                Heap Fetches: 0
                                Buffers: shared hit=525
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.011..0.012 rows=1 loops=1)
                                Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.185 ms
Execution Time: 1.889 ms
```

### bool eq 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2317.14..2317.15 rows=1 width=8) (actual time=36.373..36.397 rows=1 loops=1)
  Buffers: shared hit=18586
  ->  Limit  (cost=678.66..2316.33 rows=65 width=4) (actual time=7.359..36.225 rows=524 loops=1)
        Buffers: shared hit=18586
        ->  Result  (cost=678.66..2316.33 rows=65 width=4) (actual time=7.357..36.091 rows=524 loops=1)
              Buffers: shared hit=18586
              ->  Append  (cost=678.66..2315.68 rows=65 width=0) (actual time=7.356..35.887 rows=524 loops=1)
                    Buffers: shared hit=18586
                    ->  Subquery Scan on "*SELECT* 1"  (cost=678.66..1235.79 rows=62 width=0) (actual time=7.353..29.729 rows=512 loops=1)
                          Buffers: shared hit=15778
                          ->  Limit  (cost=678.66..1235.17 rows=62 width=4) (actual time=7.351..29.571 rows=512 loops=1)
                                Buffers: shared hit=15778
                                ->  Result  (cost=678.66..1235.17 rows=62 width=4) (actual time=7.349..29.451 rows=512 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15778
                                      ->  Hash Semi Join  (cost=678.66..1235.17 rows=62 width=0) (actual time=7.334..29.283 rows=512 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15778
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.049..13.307 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=577.54..577.54 rows=8044 width=32) (actual time=7.109..7.111 rows=8010 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 565kB
                                                  Buffers: shared hit=97
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..577.54 rows=8044 width=32) (actual time=0.031..3.560 rows=8010 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=97
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..937.04 rows=1 width=0) (actual time=0.364..5.042 rows=12 loops=1)
                          Buffers: shared hit=2460
                          ->  Limit  (cost=0.99..937.03 rows=1 width=4) (actual time=0.362..5.033 rows=12 loops=1)
                                Buffers: shared hit=2460
                                ->  Result  (cost=0.99..937.03 rows=1 width=4) (actual time=0.359..5.022 rows=12 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=2460
                                      ->  Nested Loop Semi Join  (cost=0.99..937.03 rows=1 width=0) (actual time=0.333..4.984 rows=12 loops=1)
                                            Buffers: shared hit=2460
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.079..1.533 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=484)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1937
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..96.59 rows=1 width=0) (actual time=0.681..0.684 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..96.58 rows=1 width=4) (actual time=0.680..0.682 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..96.58 rows=1 width=4) (actual time=0.677..0.678 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..96.58 rows=1 width=0) (actual time=0.658..0.659 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.079..0.235 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.94 rows=1 width=0) (actual time=0.217..0.223 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.98..45.93 rows=1 width=4) (actual time=0.216..0.221 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.98..45.93 rows=1 width=4) (actual time=0.214..0.219 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=0) (actual time=0.202..0.206 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.120..0.152 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.090..0.094 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.066..0.075 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.048..0.049 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.026..0.027 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.016..0.016 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.016..0.017 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 10.161 ms
Execution Time: 36.925 ms
```

### bool eq 2%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=15491.56..16483.98 rows=1 width=1434) (actual time=85.224..87.246 rows=51 loops=1)
  Buffers: shared hit=25459
  ->  Nested Loop  (cost=15491.26..16475.65 rows=1 width=1399) (actual time=85.189..86.513 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=25306
        ->  Subquery Scan on x  (cost=15490.98..16467.33 rows=1 width=1392) (actual time=85.164..86.024 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=25153
              ->  Limit  (cost=15490.98..16466.19 rows=51 width=1392) (actual time=85.159..85.896 rows=51 loops=1)
                    Buffers: shared hit=25153
                    ->  Merge Append  (cost=15490.98..16523.55 rows=54 width=1392) (actual time=85.156..85.876 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=25153
                          ->  Limit  (cost=15443.02..15443.15 rows=51 width=1392) (actual time=83.630..83.672 rows=50 loops=1)
                                Buffers: shared hit=24589
                                ->  Sort  (cost=15443.02..15443.18 rows=62 width=1392) (actual time=83.628..83.658 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 127kB
                                      Buffers: shared hit=24589
                                      ->  Result  (cost=678.66..15441.18 rows=62 width=1392) (actual time=6.406..82.084 rows=512 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=24589
                                            ->  Hash Semi Join  (cost=678.66..15441.18 rows=62 width=1384) (actual time=6.396..81.704 rows=512 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=24589
                                                  ->  Index Scan using records_list_group on records r  (cost=0.55..14746.00 rows=6151 width=1384) (actual time=0.075..62.729 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Buffers: shared hit=24492
                                                  ->  Hash  (cost=577.54..577.54 rows=8044 width=32) (actual time=5.768..5.771 rows=8010 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 565kB
                                                        Buffers: shared hit=97
                                                        ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..577.54 rows=8044 width=32) (actual time=0.050..2.812 rows=8010 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=97
                          ->  Limit  (cost=0.99..937.03 rows=1 width=1392) (actual time=0.422..1.029 rows=2 loops=1)
                                Buffers: shared hit=216
                                ->  Result  (cost=0.99..937.03 rows=1 width=1392) (actual time=0.419..1.024 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=216
                                      ->  Nested Loop Semi Join  (cost=0.99..937.03 rows=1 width=1384) (actual time=0.399..1.001 rows=2 loops=1)
                                            Buffers: shared hit=216
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.079..0.232 rows=42 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=47
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=42)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=169
                          ->  Limit  (cost=0.99..96.58 rows=1 width=1392) (actual time=0.861..0.865 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..96.58 rows=1 width=1392) (actual time=0.859..0.862 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..96.58 rows=1 width=1384) (actual time=0.846..0.848 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.064..0.241 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.232..0.243 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.231..0.241 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.98..45.93 rows=1 width=1392) (actual time=0.199..0.209 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=1384) (actual time=0.191..0.200 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.122..0.157 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.088..0.093 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.064..0.075 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.050..0.052 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.024..0.025 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.018..0.018 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.005..0.005 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 18.483 ms
Execution Time: 87.778 ms
```

### bool eq 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=7852.43..7852.44 rows=1 width=8) (actual time=38.438..38.443 rows=1 loops=1)
  Buffers: shared hit=32603
  ->  Limit  (cost=678.53..7839.91 rows=1001 width=4) (actual time=4.522..38.220 rows=1001 loops=1)
        Buffers: shared hit=32603
        ->  Result  (cost=678.53..15294.62 rows=2043 width=4) (actual time=4.520..37.991 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32603
              ->  Hash Semi Join  (cost=678.53..15294.62 rows=2043 width=0) (actual time=4.508..37.724 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=32603
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..14052.69 rows=203213 width=32) (actual time=0.039..19.795 rows=50170 loops=1)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=32506
                    ->  Hash  (cost=577.54..577.54 rows=8044 width=32) (actual time=4.425..4.427 rows=8010 loops=1)
                          Buckets: 8192  Batches: 1  Memory Usage: 565kB
                          Buffers: shared hit=97
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..577.54 rows=8044 width=32) (actual time=0.028..2.405 rows=8010 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true))
                                Heap Fetches: 0
                                Buffers: shared hit=97
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.366 ms
Execution Time: 38.522 ms
```

### bool eq 2%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=1027.17..1027.30 rows=51 width=1426) (actual time=101.382..101.411 rows=51 loops=1)
  Buffers: shared hit=32194
  ->  Sort  (cost=1027.17..1032.28 rows=2043 width=1426) (actual time=101.380..101.402 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 199kB
        Buffers: shared hit=32194
        ->  Result  (cost=763.17..959.01 rows=2043 width=1426) (actual time=6.916..92.226 rows=8010 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=32194
              ->  Hash Left Join  (cost=763.17..943.69 rows=2043 width=1399) (actual time=6.893..84.880 rows=8010 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32194
                    ->  Nested Loop  (cost=618.46..793.63 rows=2043 width=1391) (actual time=6.648..79.597 rows=8010 loops=1)
                          Buffers: shared hit=32140
                          ->  Nested Loop  (cost=618.04..786.34 rows=1 width=71) (actual time=6.602..12.645 rows=8010 loops=1)
                                Buffers: shared hit=100
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.025..0.029 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=617.76..697.90 rows=8014 width=32) (actual time=6.574..10.568 rows=8010 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=97
                                      ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..577.54 rows=8044 width=32) (actual time=0.034..2.493 rows=8010 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '0491d2ec-5291-4afd-907e-2a31f1c7d53c'::uuid) AND (value = true))
                                            Heap Fetches: 0
                                            Buffers: shared hit=97
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.80 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=8010)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=32040
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.227..0.228 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.047..0.196 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.033..0.165 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.015..0.015 rows=53 loops=1)
                                            Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.212 ms
Execution Time: 101.708 ms
```

### bool eq 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1191.17..1191.18 rows=1 width=8) (actual time=10.913..10.926 rows=1 loops=1)
  Buffers: shared hit=16584
  ->  Limit  (cost=80.79..1191.04 rows=10 width=4) (actual time=0.424..10.909 rows=64 loops=1)
        Buffers: shared hit=16584
        ->  Result  (cost=80.79..1191.04 rows=10 width=4) (actual time=0.423..10.898 rows=64 loops=1)
              Buffers: shared hit=16584
              ->  Append  (cost=80.79..1190.94 rows=10 width=0) (actual time=0.422..10.883 rows=64 loops=1)
                    Buffers: shared hit=16584
                    ->  Subquery Scan on "*SELECT* 1"  (cost=80.79..636.55 rows=7 width=0) (actual time=0.421..9.313 rows=63 loops=1)
                          Buffers: shared hit=15697
                          ->  Limit  (cost=80.79..636.48 rows=7 width=4) (actual time=0.421..9.300 rows=63 loops=1)
                                Buffers: shared hit=15697
                                ->  Result  (cost=80.79..636.48 rows=7 width=4) (actual time=0.420..9.290 rows=63 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=15697
                                      ->  Hash Semi Join  (cost=80.79..636.48 rows=7 width=0) (actual time=0.411..9.270 rows=63 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15697
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..539.99 rows=6151 width=32) (actual time=0.028..6.145 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=68.92..68.92 rows=905 width=32) (actual time=0.356..0.357 rows=920 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 66kB
                                                  Buffers: shared hit=16
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..68.92 rows=905 width=32) (actual time=0.019..0.199 rows=920 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=16
                    ->  Subquery Scan on "*SELECT* 2"  (cost=341.29..411.81 rows=1 width=0) (actual time=1.025..1.196 rows=1 loops=1)
                          Buffers: shared hit=539
                          ->  Limit  (cost=341.29..411.80 rows=1 width=4) (actual time=1.024..1.194 rows=1 loops=1)
                                Buffers: shared hit=539
                                ->  Result  (cost=341.29..411.80 rows=1 width=4) (actual time=1.022..1.192 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=539
                                      ->  Merge Semi Join  (cost=341.29..411.80 rows=1 width=0) (actual time=1.009..1.178 rows=1 loops=1)
                                            Merge Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=539
                                            ->  Sort  (cost=340.34..340.66 rows=131 width=32) (actual time=0.852..0.882 rows=484 loops=1)
                                                  Sort Key: r_1.id
                                                  Sort Method: quicksort  Memory: 51kB
                                                  Buffers: shared hit=523
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=32) (actual time=0.026..0.721 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..68.92 rows=905 width=32) (actual time=0.022..0.160 rows=920 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=16
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..96.59 rows=1 width=0) (actual time=0.264..0.265 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..96.58 rows=1 width=4) (actual time=0.264..0.264 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..96.58 rows=1 width=4) (actual time=0.262..0.263 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..96.58 rows=1 width=0) (actual time=0.256..0.257 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.015..0.091 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.94 rows=1 width=0) (actual time=0.090..0.092 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.98..45.93 rows=1 width=4) (actual time=0.089..0.091 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.98..45.93 rows=1 width=4) (actual time=0.087..0.089 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=0) (actual time=0.085..0.086 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.057..0.073 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.043..0.046 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.036 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.403 ms
Execution Time: 11.069 ms
```

### bool eq 0.25%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=7951.27..8063.33 rows=1 width=1434) (actual time=8.467..8.786 rows=51 loops=1)
  Buffers: shared hit=4891
  ->  Nested Loop  (cost=7950.97..8054.99 rows=1 width=1399) (actual time=8.446..8.634 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4738
        ->  Subquery Scan on x  (cost=7950.69..8046.68 rows=1 width=1392) (actual time=8.431..8.519 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4585
              ->  Limit  (cost=7950.69..8046.45 rows=10 width=1392) (actual time=8.429..8.482 rows=51 loops=1)
                    Buffers: shared hit=4585
                    ->  Merge Append  (cost=7950.69..8046.45 rows=10 width=1392) (actual time=8.427..8.475 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4585
                          ->  Limit  (cost=7487.39..7487.41 rows=7 width=1392) (actual time=6.810..6.830 rows=50 loops=1)
                                Buffers: shared hit=3697
                                ->  Sort  (cost=7487.39..7487.41 rows=7 width=1392) (actual time=6.809..6.824 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 151kB
                                      Buffers: shared hit=3697
                                      ->  Result  (cost=73.88..7487.30 rows=7 width=1392) (actual time=0.713..6.609 rows=63 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=3697
                                            ->  Nested Loop  (cost=73.88..7487.30 rows=7 width=1384) (actual time=0.704..6.577 rows=63 loops=1)
                                                  Buffers: shared hit=3697
                                                  ->  HashAggregate  (cost=73.44..82.49 rows=905 width=32) (actual time=0.633..0.847 rows=920 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 169kB
                                                        Buffers: shared hit=17
                                                        ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..68.92 rows=905 width=32) (actual time=0.045..0.256 rows=920 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=17
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.18 rows=1 width=1384) (actual time=0.006..0.006 rows=0 loops=920)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=3680
                          ->  Limit  (cost=416.33..416.33 rows=1 width=1392) (actual time=1.227..1.232 rows=1 loops=1)
                                Buffers: shared hit=540
                                ->  Sort  (cost=416.33..416.33 rows=1 width=1392) (actual time=1.225..1.229 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=540
                                      ->  Result  (cost=80.67..416.32 rows=1 width=1392) (actual time=1.002..1.211 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=540
                                            ->  Hash Semi Join  (cost=80.67..416.32 rows=1 width=1384) (actual time=0.988..1.196 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=540
                                                  ->  Index Scan using records_list_requester on records r_1  (cost=0.42..335.73 rows=131 width=1384) (actual time=0.020..0.740 rows=484 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Buffers: shared hit=523
                                                  ->  Hash  (cost=68.92..68.92 rows=905 width=32) (actual time=0.373..0.374 rows=920 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 66kB
                                                        Buffers: shared hit=17
                                                        ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v_1  (cost=0.56..68.92 rows=905 width=32) (actual time=0.029..0.195 rows=920 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=17
                          ->  Limit  (cost=0.99..96.58 rows=1 width=1392) (actual time=0.277..0.278 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..96.58 rows=1 width=1392) (actual time=0.276..0.277 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..96.58 rows=1 width=1384) (actual time=0.269..0.269 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.021..0.102 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.106..0.109 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.106..0.108 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.98..45.93 rows=1 width=1392) (actual time=0.099..0.102 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=1384) (actual time=0.096..0.098 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.066..0.080 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.052 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.044 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 7.143 ms
Execution Time: 9.143 ms
```

### bool eq 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=7483.73..7483.74 rows=1 width=8) (actual time=6.045..6.048 rows=1 loops=1)
  Buffers: shared hit=3696
  ->  Limit  (cost=73.88..7480.85 rows=230 width=4) (actual time=0.505..5.937 rows=920 loops=1)
        Buffers: shared hit=3696
        ->  Result  (cost=73.88..7480.85 rows=230 width=4) (actual time=0.504..5.812 rows=920 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=3696
              ->  Nested Loop  (cost=73.88..7480.85 rows=230 width=0) (actual time=0.496..5.663 rows=920 loops=1)
                    Buffers: shared hit=3696
                    ->  HashAggregate  (cost=73.44..82.49 rows=905 width=32) (actual time=0.473..0.648 rows=920 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 169kB
                          Buffers: shared hit=16
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..68.92 rows=905 width=32) (actual time=0.034..0.191 rows=920 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true))
                                Heap Fetches: 0
                                Buffers: shared hit=16
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.17 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=920)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=3680
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.890 ms
Execution Time: 6.108 ms
```

### bool eq 0.25%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=192.74..192.87 rows=51 width=1426) (actual time=10.355..10.373 rows=51 loops=1)
  Buffers: shared hit=6460
  ->  Sort  (cost=192.74..193.32 rows=230 width=1426) (actual time=10.353..10.366 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 201kB
        Buffers: shared hit=6460
        ->  Result  (cost=74.44..185.07 rows=230 width=1426) (actual time=0.515..9.567 rows=920 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=6460
              ->  Nested Loop Left Join  (cost=74.44..183.34 rows=230 width=1399) (actual time=0.500..8.998 rows=920 loops=1)
                    Buffers: shared hit=6460
                    ->  Nested Loop  (cost=74.14..108.08 rows=230 width=1391) (actual time=0.485..6.331 rows=920 loops=1)
                          Buffers: shared hit=3700
                          ->  Nested Loop  (cost=73.72..99.84 rows=1 width=71) (actual time=0.467..0.868 rows=920 loops=1)
                                Buffers: shared hit=20
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.023..0.026 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=73.44..82.49 rows=905 width=32) (actual time=0.443..0.649 rows=920 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 169kB
                                      Buffers: shared hit=17
                                      ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..68.92 rows=905 width=32) (actual time=0.025..0.173 rows=920 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '1e8eda75-89d7-4c4c-9d37-dd6274da6662'::uuid) AND (value = true))
                                            Heap Fetches: 0
                                            Buffers: shared hit=17
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.17 rows=1 width=1384) (actual time=0.005..0.005 rows=1 loops=920)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=3680
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=920)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=2760
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=920)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=2760
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.602 ms
Execution Time: 10.471 ms
```

### bool eq 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=675.99..676.00 rows=1 width=8) (actual time=13.153..13.184 rows=1 loops=1)
  Buffers: shared hit=391
  ->  Limit  (cost=5.93..675.94 rows=4 width=4) (actual time=0.934..13.171 rows=2 loops=1)
        Buffers: shared hit=391
        ->  Result  (cost=5.93..675.94 rows=4 width=4) (actual time=0.930..13.165 rows=2 loops=1)
              Buffers: shared hit=391
              ->  Append  (cost=5.93..675.90 rows=4 width=0) (actual time=0.928..13.160 rows=2 loops=1)
                    Buffers: shared hit=391
                    ->  Subquery Scan on "*SELECT* 1"  (cost=5.93..293.58 rows=1 width=0) (actual time=0.926..1.175 rows=2 loops=1)
                          Buffers: shared hit=141
                          ->  Limit  (cost=5.93..293.57 rows=1 width=4) (actual time=0.924..1.171 rows=2 loops=1)
                                Buffers: shared hit=141
                                ->  Result  (cost=5.93..293.57 rows=1 width=4) (actual time=0.922..1.167 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=141
                                      ->  Nested Loop  (cost=5.93..293.57 rows=1 width=0) (actual time=0.900..1.143 rows=2 loops=1)
                                            Buffers: shared hit=141
                                            ->  HashAggregate  (cost=5.49..5.83 rows=34 width=32) (actual time=0.332..0.363 rows=34 loops=1)
                                                  Group Key: v.tenant_id, v.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..5.32 rows=34 width=32) (actual time=0.275..0.290 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.022..0.022 rows=0 loops=34)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=136
                    ->  Subquery Scan on "*SELECT* 2"  (cost=5.93..293.66 rows=1 width=0) (actual time=0.634..0.640 rows=0 loops=1)
                          Buffers: shared hit=141
                          ->  Limit  (cost=5.93..293.65 rows=1 width=4) (actual time=0.632..0.637 rows=0 loops=1)
                                Buffers: shared hit=141
                                ->  Result  (cost=5.93..293.65 rows=1 width=4) (actual time=0.629..0.634 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=141
                                      ->  Nested Loop  (cost=5.93..293.65 rows=1 width=0) (actual time=0.607..0.611 rows=0 loops=1)
                                            Buffers: shared hit=141
                                            ->  HashAggregate  (cost=5.49..5.83 rows=34 width=32) (actual time=0.162..0.183 rows=34 loops=1)
                                                  Group Key: v_1.tenant_id, v_1.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..5.32 rows=34 width=32) (actual time=0.115..0.126 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=32) (actual time=0.012..0.012 rows=0 loops=34)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v_1.record_id))
                                                  Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=136
                    ->  Subquery Scan on "*SELECT* 3"  (cost=6.18..42.70 rows=1 width=0) (actual time=10.724..10.730 rows=0 loops=1)
                          Buffers: shared hit=72
                          ->  Limit  (cost=6.18..42.69 rows=1 width=4) (actual time=10.722..10.728 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Result  (cost=6.18..42.69 rows=1 width=4) (actual time=10.719..10.724 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=72
                                      ->  Hash Semi Join  (cost=6.18..42.69 rows=1 width=0) (actual time=10.705..10.710 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=72
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=32) (actual time=0.083..1.957 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=5.32..5.32 rows=34 width=32) (actual time=7.357..7.359 rows=34 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..5.32 rows=34 width=32) (actual time=0.078..4.302 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.94 rows=1 width=0) (actual time=0.533..0.538 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.98..45.93 rows=1 width=4) (actual time=0.531..0.535 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.98..45.93 rows=1 width=4) (actual time=0.527..0.532 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=0) (actual time=0.504..0.508 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.370..0.417 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.299..0.307 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.269..0.284 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.228..0.229 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.180..0.181 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.042..0.042 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.032..0.032 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.027..0.028 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 26.078 ms
Execution Time: 14.423 ms
```

### bool eq 0.01%, ACL

```sql
SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $10
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=676.54..692.70 rows=1 width=1434) (actual time=5.793..5.862 rows=2 loops=1)
  Buffers: shared hit=403
  ->  Nested Loop  (cost=676.24..684.37 rows=1 width=1399) (actual time=5.728..5.782 rows=2 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=397
        ->  Subquery Scan on x  (cost=675.96..676.06 rows=1 width=1392) (actual time=5.666..5.704 rows=2 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=391
              ->  Limit  (cost=675.96..675.97 rows=4 width=1392) (actual time=5.650..5.681 rows=2 loops=1)
                    Buffers: shared hit=391
                    ->  Sort  (cost=675.96..675.97 rows=4 width=1392) (actual time=5.647..5.675 rows=2 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Sort Method: quicksort  Memory: 29kB
                          Buffers: shared hit=391
                          ->  Append  (cost=293.58..675.92 rows=4 width=1392) (actual time=3.751..5.549 rows=2 loops=1)
                                Buffers: shared hit=391
                                ->  Limit  (cost=293.58..293.58 rows=1 width=1392) (actual time=3.746..3.754 rows=2 loops=1)
                                      Buffers: shared hit=141
                                      ->  Sort  (cost=293.58..293.58 rows=1 width=1392) (actual time=3.744..3.749 rows=2 loops=1)
                                            Sort Key: r.updated_at DESC, r.id DESC
                                            Sort Method: quicksort  Memory: 29kB
                                            Buffers: shared hit=141
                                            ->  Result  (cost=5.93..293.57 rows=1 width=1392) (actual time=3.448..3.723 rows=2 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=141
                                                  ->  Nested Loop  (cost=5.93..293.57 rows=1 width=1384) (actual time=3.386..3.656 rows=2 loops=1)
                                                        Buffers: shared hit=141
                                                        ->  HashAggregate  (cost=5.49..5.83 rows=34 width=32) (actual time=0.465..0.504 rows=34 loops=1)
                                                              Group Key: v.tenant_id, v.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=5
                                                              ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..5.32 rows=34 width=32) (actual time=0.420..0.432 rows=34 loops=1)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5
                                                        ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.083..0.083 rows=0 loops=34)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                                              Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=136
                                ->  Limit  (cost=293.66..293.67 rows=1 width=1392) (actual time=0.663..0.668 rows=0 loops=1)
                                      Buffers: shared hit=141
                                      ->  Sort  (cost=293.66..293.67 rows=1 width=1392) (actual time=0.661..0.665 rows=0 loops=1)
                                            Sort Key: r_1.updated_at DESC, r_1.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=141
                                            ->  Result  (cost=5.93..293.65 rows=1 width=1392) (actual time=0.606..0.610 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=141
                                                  ->  Nested Loop  (cost=5.93..293.65 rows=1 width=1384) (actual time=0.586..0.589 rows=0 loops=1)
                                                        Buffers: shared hit=141
                                                        ->  HashAggregate  (cost=5.49..5.83 rows=34 width=32) (actual time=0.132..0.153 rows=34 loops=1)
                                                              Group Key: v_1.tenant_id, v_1.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=5
                                                              ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v_1  (cost=0.56..5.32 rows=34 width=32) (actual time=0.086..0.098 rows=34 loops=1)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5
                                                        ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.012..0.012 rows=0 loops=34)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v_1.record_id))
                                                              Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=136
                                ->  Limit  (cost=42.70..42.70 rows=1 width=1392) (actual time=0.607..0.612 rows=0 loops=1)
                                      Buffers: shared hit=72
                                      ->  Sort  (cost=42.70..42.70 rows=1 width=1392) (actual time=0.605..0.609 rows=0 loops=1)
                                            Sort Key: r_2.updated_at DESC, r_2.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=72
                                            ->  Result  (cost=6.18..42.69 rows=1 width=1392) (actual time=0.565..0.569 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=72
                                                  ->  Hash Semi Join  (cost=6.18..42.69 rows=1 width=1384) (actual time=0.549..0.552 rows=0 loops=1)
                                                        Hash Cond: (r_2.id = v_2.record_id)
                                                        Buffers: shared hit=72
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..36.90 rows=13 width=1384) (actual time=0.129..0.333 rows=61 loops=1)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 3
                                                              Buffers: shared hit=67
                                                        ->  Hash  (cost=5.32..5.32 rows=34 width=32) (actual time=0.155..0.156 rows=34 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                              Buffers: shared hit=5
                                                              ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v_2  (cost=0.56..5.32 rows=34 width=32) (actual time=0.107..0.122 rows=34 loops=1)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5
                                ->  Limit  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.491..0.498 rows=0 loops=1)
                                      Buffers: shared hit=37
                                      ->  Sort  (cost=45.94..45.94 rows=1 width=1392) (actual time=0.489..0.496 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=37
                                            ->  Result  (cost=36.98..45.93 rows=1 width=1392) (actual time=0.456..0.462 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop Semi Join  (cost=36.98..45.93 rows=1 width=1384) (actual time=0.441..0.447 rows=0 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.203..0.260 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.130..0.139 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.107..0.121 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.087..0.089 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.051..0.052 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.032..0.032 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.034..0.034 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.45 rows=1 width=32) (actual time=0.059..0.059 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.029..0.029 rows=1 loops=2)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=6
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.024..0.024 rows=1 loops=2)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=6
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 29.413 ms
Execution Time: 8.000 ms
```

### bool eq 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=293.51..293.52 rows=1 width=8) (actual time=0.566..0.569 rows=1 loops=1)
  Buffers: shared hit=141
  ->  Limit  (cost=5.93..293.40 rows=9 width=4) (actual time=0.137..0.555 rows=34 loops=1)
        Buffers: shared hit=141
        ->  Result  (cost=5.93..293.40 rows=9 width=4) (actual time=0.134..0.544 rows=34 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=141
              ->  Nested Loop  (cost=5.93..293.40 rows=9 width=0) (actual time=0.119..0.519 rows=34 loops=1)
                    Buffers: shared hit=141
                    ->  HashAggregate  (cost=5.49..5.83 rows=34 width=32) (actual time=0.090..0.103 rows=34 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..5.32 rows=34 width=32) (actual time=0.052..0.063 rows=34 loops=1)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=34)
                          Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                          Buffers: shared hit=136
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.772 ms
Execution Time: 0.691 ms
```

### bool eq 0.01%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $9
```

```
Limit  (cost=26.08..26.10 rows=9 width=1426) (actual time=0.999..1.017 rows=34 loops=1)
  Buffers: shared hit=246
  ->  Sort  (cost=26.08..26.10 rows=9 width=1426) (actual time=0.997..1.008 rows=34 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 93kB
        Buffers: shared hit=246
        ->  Result  (cost=6.49..25.94 rows=9 width=1426) (actual time=0.212..0.927 rows=34 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
              Buffers: shared hit=246
              ->  Nested Loop Left Join  (cost=6.49..25.87 rows=9 width=1399) (actual time=0.150..0.830 rows=34 loops=1)
                    Buffers: shared hit=246
                    ->  Nested Loop  (cost=6.19..22.91 rows=9 width=1391) (actual time=0.130..0.586 rows=34 loops=1)
                          Buffers: shared hit=144
                          ->  Nested Loop  (cost=5.76..14.46 rows=1 width=71) (actual time=0.103..0.133 rows=34 loops=1)
                                Buffers: shared hit=8
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.033..0.037 rows=1 loops=1)
                                      Index Cond: (tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                                      Filter: (id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=5.49..5.83 rows=34 width=32) (actual time=0.068..0.083 rows=34 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 24kB
                                      Buffers: shared hit=5
                                      ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..5.32 rows=34 width=32) (actual time=0.032..0.042 rows=34 loops=1)
                                            Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (field_id = '4337e309-f30e-4ee6-a2c3-dac76d923042'::uuid) AND (value = true))
                                            Heap Fetches: 0
                                            Buffers: shared hit=5
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.011..0.011 rows=1 loops=34)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'd7958a18-6641-436d-ba20-684ed5abbd57'::uuid)
                                Buffers: shared hit=136
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=34)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid)
                          Buffers: shared hit=102
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=34)
                                Index Cond: ((tenant_id = '012f57b2-491a-4dae-baf2-841ce4f136ec'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=102
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.153 ms
Execution Time: 1.202 ms
```
