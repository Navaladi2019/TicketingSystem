# TICK-747 list plans (one run per cell, as nava_app)

### text eq 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=12075.34..12075.35 rows=1 width=8) (actual time=29.362..29.374 rows=1 loops=1)
  Buffers: shared hit=15892
  ->  Limit  (cost=882.71..12062.83 rows=1001 width=4) (actual time=20.899..29.229 rows=1001 loops=1)
        Buffers: shared hit=15892
        ->  Result  (cost=882.71..12397.89 rows=1031 width=4) (actual time=20.897..29.071 rows=1001 loops=1)
              Buffers: shared hit=15892
              ->  Append  (cost=882.71..12387.58 rows=1031 width=0) (actual time=20.896..28.885 rows=1001 loops=1)
                    Buffers: shared hit=15892
                    ->  Subquery Scan on "*SELECT* 1"  (cost=882.71..11035.93 rows=1001 width=0) (actual time=20.894..28.737 rows=1001 loops=1)
                          Buffers: shared hit=15892
                          ->  Limit  (cost=882.71..11025.92 rows=1001 width=4) (actual time=20.893..28.574 rows=1001 loops=1)
                                Buffers: shared hit=15892
                                ->  Result  (cost=882.71..12566.15 rows=1153 width=4) (actual time=20.892..28.442 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15892
                                      ->  Merge Semi Join  (cost=882.71..12566.15 rows=1153 width=0) (actual time=20.883..28.255 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15892
                                            ->  Sort  (cost=882.00..896.61 rows=5845 width=32) (actual time=20.811..21.416 rows=2496 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: quicksort  Memory: 2087kB
                                                  Buffers: shared hit=15681
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.031..10.170 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..11307.34 rows=157813 width=32) (actual time=0.048..4.126 rows=17379 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=211
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..1189.31 rows=27 width=0) (never executed)
                          ->  Limit  (cost=0.99..1189.04 rows=27 width=4) (never executed)
                                ->  Result  (cost=0.99..1189.04 rows=27 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.04 rows=27 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.68 rows=2 width=0) (never executed)
                          ->  Limit  (cost=0.99..111.66 rows=2 width=4) (never executed)
                                ->  Result  (cost=0.99..111.66 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..111.66 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.714 ms
Execution Time: 29.720 ms
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
Nested Loop Left Join  (cost=49.10..15519.77 rows=1 width=1434) (actual time=0.352..5.189 rows=51 loops=1)
  Buffers: shared hit=2961
  ->  Nested Loop  (cost=48.80..15511.43 rows=1 width=1399) (actual time=0.335..4.953 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2808
        ->  Subquery Scan on x  (cost=48.52..15503.12 rows=1 width=1392) (actual time=0.325..4.771 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2655
              ->  Limit  (cost=48.52..15501.97 rows=51 width=1392) (actual time=0.322..4.696 rows=51 loops=1)
                    Buffers: shared hit=2655
                    ->  Merge Append  (cost=48.52..24592.24 rows=81 width=1392) (actual time=0.321..4.682 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2655
                          ->  Limit  (cost=0.99..23244.77 rows=51 width=1392) (actual time=0.128..4.404 rows=50 loops=1)
                                Buffers: shared hit=2579
                                ->  Result  (cost=0.99..525492.64 rows=1153 width=1392) (actual time=0.127..4.392 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=2579
                                      ->  Nested Loop Semi Join  (cost=0.99..525492.64 rows=1153 width=1384) (actual time=0.122..4.330 rows=50 loops=1)
                                            Buffers: shared hit=2579
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..501702.69 rows=5845 width=1384) (actual time=0.098..3.208 rows=133 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1887
                                                  Buffers: shared hit=2046
                                            ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..4.06 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=133)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=533
                          ->  Limit  (cost=0.99..1189.04 rows=27 width=1392) (actual time=0.032..0.086 rows=2 loops=1)
                                Buffers: shared hit=29
                                ->  Result  (cost=0.99..1189.04 rows=27 width=1392) (actual time=0.032..0.085 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=29
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.04 rows=27 width=1384) (actual time=0.029..0.081 rows=2 loops=1)
                                            Buffers: shared hit=29
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.016..0.027 rows=5 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=8
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=5)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=21
                          ->  Limit  (cost=0.99..111.66 rows=2 width=1392) (actual time=0.030..0.033 rows=1 loops=1)
                                Buffers: shared hit=9
                                ->  Result  (cost=0.99..111.66 rows=2 width=1392) (actual time=0.029..0.031 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=9
                                      ->  Nested Loop Semi Join  (cost=0.99..111.66 rows=2 width=1384) (actual time=0.027..0.028 rows=1 loops=1)
                                            Buffers: shared hit=9
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.015..0.015 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=4
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=5
                          ->  Limit  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.128..0.134 rows=1 loops=1)
                                Buffers: shared hit=38
                                ->  Sort  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.128..0.133 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=38
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1392) (actual time=0.097..0.123 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=38
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1384) (actual time=0.095..0.120 rows=2 loops=1)
                                                  Buffers: shared hit=38
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.060..0.084 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.049 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.040 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.010..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.010..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.632 ms
Execution Time: 5.446 ms
```

### text eq 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=5360.67..5360.68 rows=1 width=8) (actual time=18.799..18.802 rows=1 loops=1)
  Buffers: shared hit=11142
  ->  Limit  (cost=0.99..5348.16 rows=1001 width=4) (actual time=0.066..18.575 rows=1001 loops=1)
        Buffers: shared hit=11142
        ->  Result  (cost=0.99..208764.87 rows=39081 width=4) (actual time=0.064..18.401 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=11142
              ->  Nested Loop Semi Join  (cost=0.99..208764.87 rows=39081 width=0) (actual time=0.056..18.161 rows=1001 loops=1)
                    Buffers: shared hit=11142
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.029..1.114 rows=2394 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1565
                    ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..0.97 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=2394)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r.id))
                          Heap Fetches: 0
                          Buffers: shared hit=9577
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.751 ms
Execution Time: 18.857 ms
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
Limit  (cost=1.55..925.11 rows=51 width=1426) (actual time=0.121..1.926 rows=51 loops=1)
  Buffers: shared hit=777
  ->  Result  (cost=1.55..707719.30 rows=39081 width=1426) (actual time=0.119..1.911 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
        Buffers: shared hit=777
        ->  Nested Loop Left Join  (cost=1.55..707426.19 rows=39081 width=1399) (actual time=0.099..1.845 rows=51 loops=1)
              Buffers: shared hit=777
              ->  Nested Loop  (cost=1.25..694640.86 rows=39081 width=1391) (actual time=0.082..1.560 rows=51 loops=1)
                    Buffers: shared hit=624
                    ->  Nested Loop Semi Join  (cost=0.98..694144.05 rows=39081 width=1384) (actual time=0.064..1.495 rows=51 loops=1)
                          Buffers: shared hit=621
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..500712.14 rows=198111 width=1384) (actual time=0.035..0.375 rows=123 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                Buffers: shared hit=128
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..0.97 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=123)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v40'::text COLLATE "C") AND (record_id = r.id))
                                Heap Fetches: 0
                                Buffers: shared hit=493
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.012..0.013 rows=1 loops=1)
                                Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.477 ms
Execution Time: 2.027 ms
```

### text eq 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2592.57..2592.58 rows=1 width=8) (actual time=23.507..23.519 rows=1 loops=1)
  Buffers: shared hit=18583
  ->  Limit  (cost=712.06..2591.76 rows=65 width=4) (actual time=4.212..23.259 rows=499 loops=1)
        Buffers: shared hit=18583
        ->  Result  (cost=712.06..2591.76 rows=65 width=4) (actual time=4.211..23.170 rows=499 loops=1)
              Buffers: shared hit=18583
              ->  Append  (cost=712.06..2591.11 rows=65 width=0) (actual time=4.210..23.054 rows=499 loops=1)
                    Buffers: shared hit=18583
                    ->  Subquery Scan on "*SELECT* 1"  (cost=712.06..1244.55 rows=62 width=0) (actual time=4.207..19.104 rows=493 loops=1)
                          Buffers: shared hit=15775
                          ->  Limit  (cost=712.06..1243.93 rows=62 width=4) (actual time=4.204..18.967 rows=493 loops=1)
                                Buffers: shared hit=15775
                                ->  Result  (cost=712.06..1243.93 rows=62 width=4) (actual time=4.202..18.884 rows=493 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15775
                                      ->  Hash Semi Join  (cost=712.06..1243.93 rows=62 width=0) (actual time=4.189..18.770 rows=493 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15775
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.034..9.080 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=606.16..606.16 rows=8427 width=32) (actual time=4.087..4.089 rows=7980 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 627kB
                                                  Buffers: shared hit=94
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..606.16 rows=8427 width=32) (actual time=0.028..2.294 rows=7980 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=94
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..1189.05 rows=1 width=0) (actual time=0.924..3.334 rows=6 loops=1)
                          Buffers: shared hit=2460
                          ->  Limit  (cost=0.99..1189.04 rows=1 width=4) (actual time=0.923..3.330 rows=6 loops=1)
                                Buffers: shared hit=2460
                                ->  Result  (cost=0.99..1189.04 rows=1 width=4) (actual time=0.921..3.326 rows=6 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=2460
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.04 rows=1 width=0) (actual time=0.900..3.301 rows=6 loops=1)
                                            Buffers: shared hit=2460
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=32) (actual time=0.039..0.971 rows=484 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Buffers: shared hit=523
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=484)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=1937
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.67 rows=1 width=0) (actual time=0.384..0.385 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..111.66 rows=1 width=4) (actual time=0.383..0.384 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.66 rows=1 width=4) (actual time=0.381..0.382 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.66 rows=1 width=0) (actual time=0.369..0.369 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.023..0.120 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (actual time=0.096..0.099 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (actual time=0.096..0.098 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (actual time=0.095..0.098 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (actual time=0.092..0.094 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.060..0.078 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.047..0.049 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.038 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.024..0.025 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.583 ms
Execution Time: 23.689 ms
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
Nested Loop Left Join  (cost=21055.36..22300.03 rows=1 width=1434) (actual time=71.698..72.074 rows=51 loops=1)
  Buffers: shared hit=23006
  ->  Nested Loop  (cost=21055.06..22291.70 rows=1 width=1399) (actual time=71.678..71.900 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22853
        ->  Subquery Scan on x  (cost=21054.78..22283.38 rows=1 width=1392) (actual time=71.666..71.770 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22700
              ->  Limit  (cost=21054.78..22282.24 rows=51 width=1392) (actual time=71.663..71.726 rows=51 loops=1)
                    Buffers: shared hit=22700
                    ->  Merge Append  (cost=21054.78..22354.44 rows=54 width=1392) (actual time=71.661..71.717 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22700
                          ->  Limit  (cost=21007.25..21007.37 rows=51 width=1392) (actual time=70.341..70.368 rows=51 loops=1)
                                Buffers: shared hit=21892
                                ->  Sort  (cost=21007.25..21007.40 rows=62 width=1392) (actual time=70.339..70.360 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 201kB
                                      Buffers: shared hit=21892
                                      ->  Result  (cost=1170.83..21005.40 rows=62 width=1392) (actual time=18.848..69.728 rows=493 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=21892
                                            ->  Hash Semi Join  (cost=1170.83..21005.40 rows=62 width=1384) (actual time=18.835..69.513 rows=493 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21892
                                                  ->  Bitmap Heap Scan on records r  (cost=459.32..20277.78 rows=5845 width=1384) (actual time=13.679..57.094 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.86 rows=5845 width=0) (actual time=6.197..6.198 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=606.16..606.16 rows=8427 width=32) (actual time=4.970..4.972 rows=7980 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 627kB
                                                        Buffers: shared hit=95
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..606.16 rows=8427 width=32) (actual time=0.054..2.718 rows=7980 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=95
                          ->  Limit  (cost=0.99..1189.04 rows=1 width=1392) (actual time=0.764..0.766 rows=1 loops=1)
                                Buffers: shared hit=460
                                ->  Result  (cost=0.99..1189.04 rows=1 width=1392) (actual time=0.761..0.763 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=460
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.04 rows=1 width=1384) (actual time=0.744..0.745 rows=1 loops=1)
                                            Buffers: shared hit=460
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.038..0.250 rows=90 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 5
                                                  Buffers: shared hit=99
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..4.58 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=90)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=361
                          ->  Limit  (cost=0.99..111.66 rows=1 width=1392) (actual time=0.435..0.437 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.66 rows=1 width=1392) (actual time=0.416..0.417 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.66 rows=1 width=1384) (actual time=0.409..0.410 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.019..0.138 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..4.58 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.114..0.117 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.114..0.116 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1392) (actual time=0.103..0.106 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1384) (actual time=0.100..0.102 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.063..0.083 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.047..0.050 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.039 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 4.531 ms
Execution Time: 72.374 ms
```

### text eq 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=7577.53..7577.54 rows=1 width=8) (actual time=26.430..26.436 rows=1 loops=1)
  Buffers: shared hit=31627
  ->  Limit  (cost=711.93..7565.01 rows=1001 width=4) (actual time=3.725..26.301 rows=1001 loops=1)
        Buffers: shared hit=31627
        ->  Result  (cost=711.93..15000.02 rows=2087 width=4) (actual time=3.723..26.157 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=31627
              ->  Hash Semi Join  (cost=711.93..15000.02 rows=2087 width=0) (actual time=3.716..25.972 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=31627
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.025..14.312 rows=48655 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=31533
                    ->  Hash  (cost=606.16..606.16 rows=8427 width=32) (actual time=3.654..3.656 rows=7980 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 627kB
                          Buffers: shared hit=94
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..606.16 rows=8427 width=32) (actual time=0.026..2.006 rows=7980 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=94
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.509 ms
Execution Time: 26.484 ms
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
Limit  (cost=1067.40..1067.53 rows=51 width=1426) (actual time=66.935..66.958 rows=51 loops=1)
  Buffers: shared hit=32072
  ->  Sort  (cost=1067.40..1072.62 rows=2087 width=1426) (actual time=66.934..66.951 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 187kB
        Buffers: shared hit=32072
        ->  Result  (cost=793.70..997.78 rows=2087 width=1426) (actual time=4.339..60.756 rows=7980 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32072
              ->  Hash Left Join  (cost=793.70..982.12 rows=2087 width=1399) (actual time=4.326..55.610 rows=7980 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32072
                    ->  Nested Loop  (cost=648.99..831.94 rows=2087 width=1391) (actual time=4.156..51.958 rows=7980 loops=1)
                          Buffers: shared hit=32018
                          ->  Nested Loop  (cost=648.57..824.67 rows=1 width=71) (actual time=4.135..7.923 rows=7980 loops=1)
                                Buffers: shared hit=98
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.018..0.021 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=648.29..732.33 rows=8404 width=32) (actual time=4.116..6.434 rows=7980 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=95
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..606.16 rows=8427 width=32) (actual time=0.026..1.589 rows=7980 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v2'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=95
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.76 rows=1 width=1384) (actual time=0.005..0.005 rows=1 loops=7980)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=31920
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.161..0.163 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.030..0.150 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.024..0.134 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.013..0.013 rows=53 loops=1)
                                            Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.957 ms
Execution Time: 67.080 ms
```

### text eq 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1348.26..1348.27 rows=1 width=8) (actual time=13.788..13.798 rows=1 loops=1)
  Buffers: shared hit=16360
  ->  Limit  (cost=53.28..1348.17 rows=7 width=4) (actual time=0.467..13.781 rows=46 loops=1)
        Buffers: shared hit=16360
        ->  Result  (cost=53.28..1348.17 rows=7 width=4) (actual time=0.466..13.771 rows=46 loops=1)
              Buffers: shared hit=16360
              ->  Append  (cost=53.28..1348.10 rows=7 width=0) (actual time=0.466..13.757 rows=46 loops=1)
                    Buffers: shared hit=16360
                    ->  Subquery Scan on "*SELECT* 1"  (cost=53.28..584.47 rows=4 width=0) (actual time=0.465..11.825 rows=45 loops=1)
                          Buffers: shared hit=15699
                          ->  Limit  (cost=53.28..584.43 rows=4 width=4) (actual time=0.464..11.813 rows=45 loops=1)
                                Buffers: shared hit=15699
                                ->  Result  (cost=53.28..584.43 rows=4 width=4) (actual time=0.464..11.803 rows=45 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15699
                                      ->  Hash Semi Join  (cost=53.28..584.43 rows=4 width=0) (actual time=0.456..11.782 rows=45 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15699
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.027..7.645 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.414..0.415 rows=1017 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                  Buffers: shared hit=18
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.023..0.241 rows=1017 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=18
                    ->  Subquery Scan on "*SELECT* 2"  (cost=67.27..614.47 rows=1 width=0) (actual time=1.055..1.438 rows=1 loops=1)
                          Buffers: shared hit=539
                          ->  Limit  (cost=67.27..614.46 rows=1 width=4) (actual time=1.052..1.435 rows=1 loops=1)
                                Buffers: shared hit=539
                                ->  Result  (cost=67.27..614.46 rows=1 width=4) (actual time=1.051..1.433 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=539
                                      ->  Hash Semi Join  (cost=67.27..614.46 rows=1 width=0) (actual time=1.042..1.424 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=539
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.177..0.956 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.096..0.096 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.397..0.397 rows=1017 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                  Buffers: shared hit=18
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..45.51 rows=576 width=32) (actual time=0.026..0.230 rows=1017 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=18
                    ->  Subquery Scan on "*SELECT* 3"  (cost=57.38..103.61 rows=1 width=0) (actual time=0.395..0.396 rows=0 loops=1)
                          Buffers: shared hit=85
                          ->  Limit  (cost=57.38..103.60 rows=1 width=4) (actual time=0.394..0.395 rows=0 loops=1)
                                Buffers: shared hit=85
                                ->  Result  (cost=57.38..103.60 rows=1 width=4) (actual time=0.393..0.394 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=85
                                      ->  Merge Semi Join  (cost=57.38..103.60 rows=1 width=0) (actual time=0.386..0.387 rows=0 loops=1)
                                            Merge Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=85
                                            ->  Sort  (cost=56.81..56.84 rows=12 width=32) (actual time=0.123..0.127 rows=61 loops=1)
                                                  Sort Key: r_2.id
                                                  Sort Method: quicksort  Memory: 28kB
                                                  Buffers: shared hit=67
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.017..0.099 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..45.51 rows=576 width=32) (actual time=0.015..0.171 rows=1016 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=18
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (actual time=0.081..0.083 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (actual time=0.080..0.082 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (actual time=0.079..0.081 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (actual time=0.077..0.079 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.048..0.064 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.036..0.038 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.025..0.029 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.963 ms
Execution Time: 13.938 ms
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
Nested Loop Left Join  (cost=5598.57..5614.92 rows=1 width=1434) (actual time=8.988..9.314 rows=46 loops=1)
  Buffers: shared hit=5026
  ->  Nested Loop  (cost=5598.27..5606.59 rows=1 width=1399) (actual time=8.972..9.167 rows=46 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4888
        ->  Subquery Scan on x  (cost=5597.99..5598.28 rows=1 width=1392) (actual time=8.961..9.036 rows=46 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4750
              ->  Limit  (cost=5597.99..5598.12 rows=7 width=1392) (actual time=8.959..9.000 rows=46 loops=1)
                    Buffers: shared hit=4750
                    ->  Merge Append  (cost=5597.99..5598.12 rows=7 width=1392) (actual time=8.957..8.993 rows=46 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4750
                          ->  Limit  (cost=4828.60..4828.61 rows=4 width=1392) (actual time=6.752..6.768 rows=45 loops=1)
                                Buffers: shared hit=4087
                                ->  Sort  (cost=4828.60..4828.61 rows=4 width=1392) (actual time=6.751..6.762 rows=45 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 115kB
                                      Buffers: shared hit=4087
                                      ->  Result  (cost=48.83..4828.56 rows=4 width=1392) (actual time=0.692..6.689 rows=45 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=4087
                                            ->  Nested Loop  (cost=48.83..4828.56 rows=4 width=1384) (actual time=0.687..6.667 rows=45 loops=1)
                                                  Buffers: shared hit=4087
                                                  ->  HashAggregate  (cost=48.39..54.15 rows=576 width=32) (actual time=0.583..0.834 rows=1017 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=19
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.037..0.235 rows=1017 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=19
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.28 rows=1 width=1384) (actual time=0.005..0.005 rows=0 loops=1017)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=4068
                          ->  Limit  (cost=614.47..614.47 rows=1 width=1392) (actual time=1.510..1.514 rows=1 loops=1)
                                Buffers: shared hit=540
                                ->  Sort  (cost=614.47..614.47 rows=1 width=1392) (actual time=1.508..1.511 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=540
                                      ->  Result  (cost=67.27..614.46 rows=1 width=1392) (actual time=1.123..1.500 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=540
                                            ->  Hash Semi Join  (cost=67.27..614.46 rows=1 width=1384) (actual time=1.114..1.489 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=540
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.184..0.999 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.098..0.098 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.411..0.411 rows=1017 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                        Buffers: shared hit=19
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_1  (cost=0.55..45.51 rows=576 width=32) (actual time=0.022..0.225 rows=1017 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=19
                          ->  Limit  (cost=109.36..109.37 rows=1 width=1392) (actual time=0.605..0.607 rows=0 loops=1)
                                Buffers: shared hit=86
                                ->  Sort  (cost=109.36..109.37 rows=1 width=1392) (actual time=0.604..0.606 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=86
                                      ->  Result  (cost=53.15..109.35 rows=1 width=1392) (actual time=0.598..0.599 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=86
                                            ->  Hash Semi Join  (cost=53.15..109.35 rows=1 width=1384) (actual time=0.591..0.592 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=86
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.017..0.107 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.471..0.472 rows=1017 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 72kB
                                                        Buffers: shared hit=19
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_2  (cost=0.55..45.51 rows=576 width=32) (actual time=0.017..0.217 rows=1017 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=19
                          ->  Limit  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.084..0.087 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.083..0.086 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1392) (actual time=0.081..0.083 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1384) (actual time=0.077..0.079 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.048..0.064 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.035..0.038 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.026..0.030 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=46)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=138
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=46)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=138
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.359 ms
Execution Time: 9.574 ms
```

### text eq 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=4827.47..4827.48 rows=1 width=8) (actual time=7.338..7.341 rows=1 loops=1)
  Buffers: shared hit=4022
  ->  Limit  (cost=48.83..4825.69 rows=143 width=4) (actual time=1.084..7.197 rows=1001 loops=1)
        Buffers: shared hit=4022
        ->  Result  (cost=48.83..4825.69 rows=143 width=4) (actual time=1.082..7.048 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=4022
              ->  Nested Loop  (cost=48.83..4825.69 rows=143 width=0) (actual time=1.076..6.870 rows=1001 loops=1)
                    Buffers: shared hit=4022
                    ->  HashAggregate  (cost=48.39..54.15 rows=576 width=32) (actual time=1.041..1.253 rows=1001 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=18
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.114..0.450 rows=1017 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=18
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=1001)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=4004
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.609 ms
Execution Time: 7.394 ms
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
Limit  (cost=129.16..129.29 rows=51 width=1426) (actual time=12.912..12.930 rows=51 loops=1)
  Buffers: shared hit=7141
  ->  Sort  (cost=129.16..129.52 rows=143 width=1426) (actual time=12.909..12.922 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 201kB
        Buffers: shared hit=7141
        ->  Result  (cost=49.39..124.39 rows=143 width=1426) (actual time=1.061..11.974 rows=1017 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=7141
              ->  Nested Loop Left Join  (cost=49.39..123.32 rows=143 width=1399) (actual time=1.043..11.268 rows=1017 loops=1)
                    Buffers: shared hit=7141
                    ->  Nested Loop  (cost=49.09..76.52 rows=143 width=1391) (actual time=1.025..7.940 rows=1017 loops=1)
                          Buffers: shared hit=4090
                          ->  Nested Loop  (cost=48.67..68.21 rows=1 width=71) (actual time=1.005..1.522 rows=1017 loops=1)
                                Buffers: shared hit=22
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.031..0.034 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=48.39..54.15 rows=576 width=32) (actual time=0.972..1.236 rows=1017 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=19
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.036..0.338 rows=1017 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v025'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=19
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=1384) (actual time=0.006..0.006 rows=1 loops=1017)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=4068
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=1017)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=3051
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=1017)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=3051
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.661 ms
Execution Time: 13.084 ms
```

### text eq 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::text COLLATE "C") AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1348.26..1348.27 rows=1 width=8) (actual time=12.117..12.130 rows=1 loops=1)
  Buffers: shared hit=16321
  ->  Limit  (cost=53.28..1348.17 rows=7 width=4) (actual time=7.437..12.121 rows=2 loops=1)
        Buffers: shared hit=16321
        ->  Result  (cost=53.28..1348.17 rows=7 width=4) (actual time=7.436..12.119 rows=2 loops=1)
              Buffers: shared hit=16321
              ->  Append  (cost=53.28..1348.10 rows=7 width=0) (actual time=7.435..12.117 rows=2 loops=1)
                    Buffers: shared hit=16321
                    ->  Subquery Scan on "*SELECT* 1"  (cost=53.28..584.47 rows=4 width=0) (actual time=7.433..10.555 rows=2 loops=1)
                          Buffers: shared hit=15686
                          ->  Limit  (cost=53.28..584.43 rows=4 width=4) (actual time=7.431..10.551 rows=2 loops=1)
                                Buffers: shared hit=15686
                                ->  Result  (cost=53.28..584.43 rows=4 width=4) (actual time=7.430..10.549 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15686
                                      ->  Hash Semi Join  (cost=53.28..584.43 rows=4 width=0) (actual time=7.421..10.538 rows=2 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15686
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.030..7.695 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.044..0.044 rows=48 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.023..0.031 rows=48 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 2"  (cost=67.27..614.47 rows=1 width=0) (actual time=1.260..1.263 rows=0 loops=1)
                          Buffers: shared hit=526
                          ->  Limit  (cost=67.27..614.46 rows=1 width=4) (actual time=1.259..1.261 rows=0 loops=1)
                                Buffers: shared hit=526
                                ->  Result  (cost=67.27..614.46 rows=1 width=4) (actual time=1.257..1.259 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=526
                                      ->  Hash Semi Join  (cost=67.27..614.46 rows=1 width=0) (actual time=1.243..1.245 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=526
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.256..1.132 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.145..0.145 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.047..0.047 rows=48 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..45.51 rows=576 width=32) (actual time=0.025..0.034 rows=48 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 3"  (cost=57.38..103.61 rows=1 width=0) (actual time=0.194..0.195 rows=0 loops=1)
                          Buffers: shared hit=72
                          ->  Limit  (cost=57.38..103.60 rows=1 width=4) (actual time=0.192..0.193 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Result  (cost=57.38..103.60 rows=1 width=4) (actual time=0.191..0.192 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=72
                                      ->  Merge Semi Join  (cost=57.38..103.60 rows=1 width=0) (actual time=0.184..0.184 rows=0 loops=1)
                                            Merge Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=72
                                            ->  Sort  (cost=56.81..56.84 rows=12 width=32) (actual time=0.144..0.149 rows=61 loops=1)
                                                  Sort Key: r_2.id
                                                  Sort Method: quicksort  Memory: 28kB
                                                  Buffers: shared hit=67
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.018..0.116 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_2  (cost=0.55..45.51 rows=576 width=32) (actual time=0.014..0.020 rows=48 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.97..45.51 rows=1 width=0) (actual time=0.093..0.096 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.97..45.50 rows=1 width=4) (actual time=0.092..0.095 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.97..45.50 rows=1 width=4) (actual time=0.092..0.094 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=0) (actual time=0.089..0.091 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.057..0.075 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.044 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.030..0.035 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.024 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C") AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.441 ms
Execution Time: 12.290 ms
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
Nested Loop Left Join  (cost=5598.57..5614.92 rows=1 width=1434) (actual time=3.314..3.359 rows=2 loops=1)
  Buffers: shared hit=844
  ->  Nested Loop  (cost=5598.27..5606.59 rows=1 width=1399) (actual time=3.286..3.322 rows=2 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=838
        ->  Subquery Scan on x  (cost=5597.99..5598.28 rows=1 width=1392) (actual time=3.266..3.293 rows=2 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=832
              ->  Limit  (cost=5597.99..5598.12 rows=7 width=1392) (actual time=3.259..3.283 rows=2 loops=1)
                    Buffers: shared hit=832
                    ->  Merge Append  (cost=5597.99..5598.12 rows=7 width=1392) (actual time=3.257..3.280 rows=2 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=832
                          ->  Limit  (cost=4828.60..4828.61 rows=4 width=1392) (actual time=0.708..0.715 rows=2 loops=1)
                                Buffers: shared hit=197
                                ->  Sort  (cost=4828.60..4828.61 rows=4 width=1392) (actual time=0.707..0.713 rows=2 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=197
                                      ->  Result  (cost=48.83..4828.56 rows=4 width=1392) (actual time=0.255..0.697 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=197
                                            ->  Nested Loop  (cost=48.83..4828.56 rows=4 width=1384) (actual time=0.249..0.688 rows=2 loops=1)
                                                  Buffers: shared hit=197
                                                  ->  HashAggregate  (cost=48.39..54.15 rows=576 width=32) (actual time=0.095..0.121 rows=48 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 49kB
                                                        Buffers: shared hit=5
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.051..0.063 rows=48 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.28 rows=1 width=1384) (actual time=0.011..0.011 rows=0 loops=48)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=192
                          ->  Limit  (cost=614.47..614.47 rows=1 width=1392) (actual time=1.901..1.906 rows=0 loops=1)
                                Buffers: shared hit=526
                                ->  Sort  (cost=614.47..614.47 rows=1 width=1392) (actual time=1.899..1.904 rows=0 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=526
                                      ->  Result  (cost=67.27..614.46 rows=1 width=1392) (actual time=1.893..1.898 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=526
                                            ->  Hash Semi Join  (cost=67.27..614.46 rows=1 width=1384) (actual time=1.889..1.893 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=526
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.373..1.737 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.225..0.226 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.054..0.056 rows=48 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                        Buffers: shared hit=5
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_1  (cost=0.55..45.51 rows=576 width=32) (actual time=0.023..0.036 rows=48 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                          ->  Limit  (cost=109.36..109.37 rows=1 width=1392) (actual time=0.465..0.467 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Sort  (cost=109.36..109.37 rows=1 width=1392) (actual time=0.463..0.465 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=72
                                      ->  Result  (cost=53.15..109.35 rows=1 width=1392) (actual time=0.454..0.456 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=72
                                            ->  Hash Semi Join  (cost=53.15..109.35 rows=1 width=1384) (actual time=0.445..0.447 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=72
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.027..0.195 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=45.51..45.51 rows=576 width=32) (actual time=0.231..0.231 rows=48 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                        Buffers: shared hit=5
                                                        ->  Index Only Scan Backward using rfv_text_value on rfv_text v_2  (cost=0.55..45.51 rows=576 width=32) (actual time=0.136..0.151 rows=48 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=5
                          ->  Limit  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.179..0.184 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.51..45.52 rows=1 width=1392) (actual time=0.178..0.183 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.97..45.50 rows=1 width=1392) (actual time=0.172..0.177 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.97..45.50 rows=1 width=1384) (actual time=0.167..0.170 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.083..0.113 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.060..0.064 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.045..0.053 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.020..0.020 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.013..0.013 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_3  (cost=0.55..1.03 rows=1 width=32) (actual time=0.017..0.017 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C") AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.007..0.008 rows=1 loops=2)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=6
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=2)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=6
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.837 ms
Execution Time: 3.734 ms
```

### text eq 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::text COLLATE "C") LIMIT 1001) c
```

```
Aggregate  (cost=4827.47..4827.48 rows=1 width=8) (actual time=0.601..0.605 rows=1 loops=1)
  Buffers: shared hit=197
  ->  Limit  (cost=48.83..4825.69 rows=143 width=4) (actual time=0.124..0.593 rows=48 loops=1)
        Buffers: shared hit=197
        ->  Result  (cost=48.83..4825.69 rows=143 width=4) (actual time=0.123..0.584 rows=48 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=197
              ->  Nested Loop  (cost=48.83..4825.69 rows=143 width=0) (actual time=0.092..0.545 rows=48 loops=1)
                    Buffers: shared hit=197
                    ->  HashAggregate  (cost=48.39..54.15 rows=576 width=32) (actual time=0.067..0.084 rows=48 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 49kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.039..0.048 rows=48 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=48)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=192
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.576 ms
Execution Time: 0.680 ms
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
Limit  (cost=129.16..129.29 rows=51 width=1426) (actual time=0.697..0.707 rows=48 loops=1)
  Buffers: shared hit=344
  ->  Sort  (cost=129.16..129.52 rows=143 width=1426) (actual time=0.695..0.701 rows=48 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 121kB
        Buffers: shared hit=344
        ->  Result  (cost=49.39..124.39 rows=143 width=1426) (actual time=0.101..0.651 rows=48 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=344
              ->  Nested Loop Left Join  (cost=49.39..123.32 rows=143 width=1399) (actual time=0.089..0.614 rows=48 loops=1)
                    Buffers: shared hit=344
                    ->  Nested Loop  (cost=49.09..76.52 rows=143 width=1391) (actual time=0.077..0.450 rows=48 loops=1)
                          Buffers: shared hit=200
                          ->  Nested Loop  (cost=48.67..68.21 rows=1 width=71) (actual time=0.061..0.085 rows=48 loops=1)
                                Buffers: shared hit=8
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.020..0.022 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=48.39..54.15 rows=576 width=32) (actual time=0.041..0.054 rows=48 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 49kB
                                      Buffers: shared hit=5
                                      ->  Index Only Scan Backward using rfv_text_value on rfv_text v  (cost=0.55..45.51 rows=576 width=32) (actual time=0.021..0.027 rows=48 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'f044d22a-32c0-4c24-9e15-62d4f3d1f65e'::uuid) AND (value = 'v001'::text COLLATE "C"))
                                            Heap Fetches: 0
                                            Buffers: shared hit=5
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.27 rows=1 width=1384) (actual time=0.006..0.006 rows=1 loops=48)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=192
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=48)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=144
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=48)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=144
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.762 ms
Execution Time: 0.777 ms
```

### text in (2 values) 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17697.96..17697.97 rows=1 width=8) (actual time=87.484..87.493 rows=1 loops=1)
  Buffers: shared hit=5055, temp written=526
  I/O Timings: temp write=4.311
  ->  Limit  (cost=14377.33..17685.45 rows=1001 width=4) (actual time=83.313..87.383 rows=1001 loops=1)
        Buffers: shared hit=5055, temp written=526
        I/O Timings: temp write=4.311
        ->  Result  (cost=14377.33..17784.59 rows=1031 width=4) (actual time=83.312..87.260 rows=1001 loops=1)
              Buffers: shared hit=5055, temp written=526
              I/O Timings: temp write=4.311
              ->  Append  (cost=14377.33..17774.28 rows=1031 width=0) (actual time=83.312..87.104 rows=1001 loops=1)
                    Buffers: shared hit=5055, temp written=526
                    I/O Timings: temp write=4.311
                    ->  Subquery Scan on "*SELECT* 1"  (cost=14377.33..15852.75 rows=1001 width=0) (actual time=83.310..86.977 rows=1001 loops=1)
                          Buffers: shared hit=5055, temp written=526
                          I/O Timings: temp write=4.311
                          ->  Limit  (cost=14377.33..15842.74 rows=1001 width=4) (actual time=83.309..86.844 rows=1001 loops=1)
                                Buffers: shared hit=5055, temp written=526
                                I/O Timings: temp write=4.311
                                ->  Result  (cost=14377.33..16112.10 rows=1185 width=4) (actual time=83.308..86.747 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=5055, temp written=526
                                      I/O Timings: temp write=4.311
                                      ->  Hash Semi Join  (cost=14377.33..16112.10 rows=1185 width=0) (actual time=83.300..86.600 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=5055, temp written=526
                                            I/O Timings: temp write=4.311
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.026..1.535 rows=4791 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=3117
                                            ->  Hash  (cost=11241.11..11241.11 rows=162133 width=32) (actual time=82.750..82.751 rows=160142 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 6000kB
                                                  Buffers: shared hit=1938, temp written=511
                                                  I/O Timings: temp write=4.187
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..11241.11 rows=162133 width=32) (actual time=0.027..37.351 rows=160142 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p40a,p40b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1938
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1712.44 rows=27 width=0) (never executed)
                          ->  Limit  (cost=0.86..1712.17 rows=27 width=4) (never executed)
                                ->  Result  (cost=0.86..1712.17 rows=27 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1712.17 rows=27 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_1  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_1.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.18 rows=2 width=0) (never executed)
                          ->  Limit  (cost=0.86..158.16 rows=2 width=4) (never executed)
                                ->  Result  (cost=0.86..158.16 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.76 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.75 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.75 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.327 ms
Execution Time: 88.562 ms
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
Nested Loop Left Join  (cost=48.98..16001.51 rows=1 width=1434) (actual time=0.470..4.798 rows=51 loops=1)
  Buffers: shared hit=2911
  ->  Nested Loop  (cost=48.68..15993.18 rows=1 width=1399) (actual time=0.450..4.583 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2758
        ->  Subquery Scan on x  (cost=48.40..15984.86 rows=1 width=1392) (actual time=0.438..4.392 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2605
              ->  Limit  (cost=48.40..15983.72 rows=51 width=1392) (actual time=0.436..4.329 rows=51 loops=1)
                    Buffers: shared hit=2605
                    ->  Merge Append  (cost=48.40..25357.43 rows=81 width=1392) (actual time=0.435..4.317 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2605
                          ->  Limit  (cost=0.86..23440.08 rows=51 width=1392) (actual time=0.167..3.981 rows=50 loops=1)
                                Buffers: shared hit=2501
                                ->  Result  (cost=0.86..544618.02 rows=1185 width=1392) (actual time=0.166..3.970 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=2501
                                      ->  Nested Loop Semi Join  (cost=0.86..544618.02 rows=1185 width=1384) (actual time=0.162..3.943 rows=50 loops=1)
                                            Buffers: shared hit=2501
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..501702.69 rows=5845 width=1384) (actual time=0.086..3.028 rows=130 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1826
                                                  Buffers: shared hit=1981
                                            ->  Index Scan using rfv_text_pkey on rfv_text v  (cost=0.43..7.33 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=130)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=520
                          ->  Limit  (cost=0.86..1712.17 rows=27 width=1392) (actual time=0.046..0.088 rows=2 loops=1)
                                Buffers: shared hit=28
                                ->  Result  (cost=0.86..1712.17 rows=27 width=1392) (actual time=0.046..0.087 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=28
                                      ->  Nested Loop Semi Join  (cost=0.86..1712.17 rows=27 width=1384) (actual time=0.043..0.084 rows=2 loops=1)
                                            Buffers: shared hit=28
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.019..0.028 rows=5 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=8
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_1  (cost=0.43..8.45 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=5)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_1.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=20
                          ->  Limit  (cost=0.86..158.16 rows=2 width=1392) (actual time=0.098..0.101 rows=1 loops=1)
                                Buffers: shared hit=39
                                ->  Result  (cost=0.86..158.16 rows=2 width=1392) (actual time=0.098..0.100 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=39
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=2 width=1384) (actual time=0.096..0.097 rows=1 loops=1)
                                            Buffers: shared hit=39
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.017..0.033 rows=7 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=11
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=7)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=28
                          ->  Limit  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.122..0.127 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.121..0.127 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.75 rows=1 width=1392) (actual time=0.076..0.116 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=1384) (actual time=0.074..0.112 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.058..0.077 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.045 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.039 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                        Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.872 ms
Execution Time: 5.131 ms
```

### text in (2 values) 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=6241.27..6241.28 rows=1 width=8) (actual time=14.647..14.650 rows=1 loops=1)
  Buffers: shared hit=11330
  ->  Limit  (cost=0.86..6228.76 rows=1001 width=4) (actual time=0.078..14.475 rows=1001 loops=1)
        Buffers: shared hit=11330
        ->  Result  (cost=0.86..249800.99 rows=40150 width=4) (actual time=0.077..14.290 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=11330
              ->  Nested Loop Semi Join  (cost=0.86..249800.99 rows=40150 width=0) (actual time=0.068..14.051 rows=1001 loops=1)
                    Buffers: shared hit=11330
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.028..1.041 rows=2436 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1586
                    ->  Index Scan using rfv_text_pkey on rfv_text v  (cost=0.43..1.18 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2436)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                          Filter: (value = ANY ('{p40a,p40b}'::text[]))
                          Rows Removed by Filter: 1
                          Buffers: shared hit=9744
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.862 ms
Execution Time: 14.707 ms
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
Limit  (cost=1.43..953.00 rows=51 width=1426) (actual time=0.158..1.297 rows=51 loops=1)
  Buffers: shared hit=681
  ->  Result  (cost=1.43..749137.22 rows=40150 width=1426) (actual time=0.157..1.286 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
        Buffers: shared hit=681
        ->  Nested Loop Left Join  (cost=1.43..748836.09 rows=40150 width=1399) (actual time=0.129..1.225 rows=51 loops=1)
              Buffers: shared hit=681
              ->  Nested Loop  (cost=1.12..735701.04 rows=40150 width=1391) (actual time=0.114..0.984 rows=51 loops=1)
                    Buffers: shared hit=528
                    ->  Nested Loop Semi Join  (cost=0.85..735190.87 rows=40150 width=1384) (actual time=0.097..0.932 rows=51 loops=1)
                          Buffers: shared hit=525
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..500712.14 rows=198111 width=1384) (actual time=0.028..0.223 rows=104 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                Buffers: shared hit=109
                          ->  Index Scan using rfv_text_pkey on rfv_text v  (cost=0.43..1.18 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=104)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                Filter: (value = ANY ('{p40a,p40b}'::text[]))
                                Rows Removed by Filter: 1
                                Buffers: shared hit=416
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.012..0.013 rows=1 loops=1)
                                Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.155 ms
Execution Time: 1.383 ms
```

### text in (2 values) 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2857.19..2857.20 rows=1 width=8) (actual time=18.479..18.488 rows=1 loops=1)
  Buffers: shared hit=16752
  ->  Limit  (cost=778.74..2856.30 rows=71 width=4) (actual time=3.376..18.423 rows=491 loops=1)
        Buffers: shared hit=16752
        ->  Result  (cost=778.74..2856.30 rows=71 width=4) (actual time=3.374..18.355 rows=491 loops=1)
              Buffers: shared hit=16752
              ->  Append  (cost=778.74..2855.59 rows=71 width=0) (actual time=3.373..18.267 rows=491 loops=1)
                    Buffers: shared hit=16752
                    ->  Subquery Scan on "*SELECT* 1"  (cost=778.74..1311.34 rows=67 width=0) (actual time=3.372..13.436 rows=479 loops=1)
                          Buffers: shared hit=15782
                          ->  Limit  (cost=778.74..1310.67 rows=67 width=4) (actual time=3.371..13.362 rows=479 loops=1)
                                Buffers: shared hit=15782
                                ->  Result  (cost=778.74..1310.67 rows=67 width=4) (actual time=3.370..13.302 rows=479 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15782
                                      ->  Hash Semi Join  (cost=778.74..1310.67 rows=67 width=0) (actual time=3.363..13.223 rows=479 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15782
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.026..6.566 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=663.51..663.51 rows=9173 width=32) (actual time=3.275..3.275 rows=7997 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 628kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..663.51 rows=9173 width=32) (actual time=0.024..1.815 rows=7997 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 2"  (cost=792.73..1339.96 rows=2 width=0) (actual time=3.297..4.124 rows=9 loops=1)
                          Buffers: shared hit=622
                          ->  Limit  (cost=792.73..1339.94 rows=2 width=4) (actual time=3.296..4.121 rows=9 loops=1)
                                Buffers: shared hit=622
                                ->  Result  (cost=792.73..1339.94 rows=2 width=4) (actual time=3.295..4.119 rows=9 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=622
                                      ->  Hash Semi Join  (cost=792.73..1339.94 rows=2 width=0) (actual time=3.286..4.108 rows=9 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=622
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.187..1.002 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.101..0.101 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=663.51..663.51 rows=9173 width=32) (actual time=3.007..3.008 rows=7997 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 628kB
                                                  Buffers: shared hit=101
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..663.51 rows=9173 width=32) (actual time=0.022..1.689 rows=7997 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=101
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.17 rows=1 width=0) (actual time=0.165..0.538 rows=3 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..158.16 rows=1 width=4) (actual time=0.165..0.537 rows=3 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=4) (actual time=0.163..0.535 rows=3 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=0) (actual time=0.157..0.528 rows=3 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.021..0.112 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.76 rows=1 width=0) (actual time=0.090..0.092 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.75 rows=1 width=4) (actual time=0.089..0.091 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.75 rows=1 width=4) (actual time=0.088..0.090 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=0) (actual time=0.085..0.086 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.050..0.065 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.040 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.025..0.030 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.019..0.020 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.210 ms
Execution Time: 18.626 ms
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
Nested Loop Left Join  (cost=22461.37..22625.33 rows=1 width=1434) (actual time=55.610..55.958 rows=51 loops=1)
  Buffers: shared hit=22937
  ->  Nested Loop  (cost=22461.08..22617.00 rows=1 width=1399) (actual time=55.587..55.798 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22784
        ->  Subquery Scan on x  (cost=22460.79..22608.69 rows=1 width=1392) (actual time=55.571..55.660 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22631
              ->  Limit  (cost=22460.79..22607.54 rows=51 width=1392) (actual time=55.568..55.622 rows=51 loops=1)
                    Buffers: shared hit=22631
                    ->  Merge Append  (cost=22460.79..22619.05 rows=55 width=1392) (actual time=55.566..55.614 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22631
                          ->  Limit  (cost=21074.17..21074.30 rows=51 width=1392) (actual time=50.897..50.920 rows=50 loops=1)
                                Buffers: shared hit=21898
                                ->  Sort  (cost=21074.17..21074.34 rows=67 width=1392) (actual time=50.896..50.914 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 193kB
                                      Buffers: shared hit=21898
                                      ->  Result  (cost=1237.51..21072.14 rows=67 width=1392) (actual time=12.566..50.479 rows=479 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=21898
                                            ->  Hash Semi Join  (cost=1237.51..21072.14 rows=67 width=1384) (actual time=12.560..50.323 rows=479 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21898
                                                  ->  Bitmap Heap Scan on records r  (cost=459.32..20277.78 rows=5845 width=1384) (actual time=9.005..42.289 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.86 rows=5845 width=0) (actual time=4.562..4.563 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=663.51..663.51 rows=9173 width=32) (actual time=3.519..3.520 rows=7997 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 628kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..663.51 rows=9173 width=32) (actual time=0.031..2.052 rows=7997 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=1339.95..1339.96 rows=2 width=1392) (actual time=4.305..4.309 rows=2 loops=1)
                                Buffers: shared hit=622
                                ->  Sort  (cost=1339.95..1339.96 rows=2 width=1392) (actual time=4.303..4.306 rows=2 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 43kB
                                      Buffers: shared hit=622
                                      ->  Result  (cost=792.73..1339.94 rows=2 width=1392) (actual time=3.388..4.272 rows=9 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=622
                                            ->  Hash Semi Join  (cost=792.73..1339.94 rows=2 width=1384) (actual time=3.375..4.256 rows=9 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=622
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.179..1.025 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.100..0.101 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=663.51..663.51 rows=9173 width=32) (actual time=3.122..3.123 rows=7997 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 628kB
                                                        Buffers: shared hit=101
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..663.51 rows=9173 width=32) (actual time=0.028..1.703 rows=7997 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=101
                          ->  Limit  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.220..0.221 rows=1 loops=1)
                                Buffers: shared hit=74
                                ->  Result  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.218..0.219 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=74
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=1384) (actual time=0.206..0.207 rows=1 loops=1)
                                            Buffers: shared hit=74
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.032..0.057 rows=14 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=18
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=14)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p2a,p2b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=56
                          ->  Limit  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.135..0.138 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.134..0.137 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.75 rows=1 width=1392) (actual time=0.125..0.128 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=1384) (actual time=0.122..0.124 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.077..0.095 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.059..0.062 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.044..0.049 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.036..0.037 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.014..0.014 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.010 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
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
Planning Time: 2.679 ms
Execution Time: 56.192 ms
```

### text in (2 values) 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=7087.19..7087.20 rows=1 width=8) (actual time=25.230..25.236 rows=1 loops=1)
  Buffers: shared hit=32719
  ->  Limit  (cost=778.61..7074.68 rows=1001 width=4) (actual time=3.443..25.097 rows=1001 loops=1)
        Buffers: shared hit=32719
        ->  Result  (cost=778.61..15068.99 rows=2272 width=4) (actual time=3.441..24.961 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32719
              ->  Hash Semi Join  (cost=778.61..15068.99 rows=2272 width=0) (actual time=3.436..24.790 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=32719
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.023..13.848 rows=50351 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=32618
                    ->  Hash  (cost=663.51..663.51 rows=9173 width=32) (actual time=3.366..3.369 rows=7997 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 628kB
                          Buffers: shared hit=101
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..663.51 rows=9173 width=32) (actual time=0.027..1.930 rows=7997 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                Heap Fetches: 0
                                Buffers: shared hit=101
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.541 ms
Execution Time: 25.288 ms
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
Limit  (cost=1151.29..1151.42 rows=51 width=1426) (actual time=68.446..68.470 rows=51 loops=1)
  Buffers: shared hit=32146
  ->  Sort  (cost=1151.29..1156.97 rows=2272 width=1426) (actual time=68.444..68.462 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 205kB
        Buffers: shared hit=32146
        ->  Result  (cost=854.78..1075.49 rows=2272 width=1426) (actual time=4.655..62.269 rows=7997 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32146
              ->  Hash Left Join  (cost=854.78..1058.45 rows=2272 width=1399) (actual time=4.637..57.121 rows=7997 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32146
                    ->  Nested Loop  (cost=710.07..907.79 rows=2272 width=1391) (actual time=4.452..53.420 rows=7997 loops=1)
                          Buffers: shared hit=32092
                          ->  Nested Loop  (cost=709.65..900.57 rows=1 width=71) (actual time=4.423..8.288 rows=7997 loops=1)
                                Buffers: shared hit=104
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.022..0.026 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=709.38..800.83 rows=9145 width=32) (actual time=4.398..6.781 rows=7997 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=101
                                      ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..663.51 rows=9173 width=32) (actual time=0.031..1.784 rows=7997 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p2a,p2b}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=101
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.67 rows=1 width=1384) (actual time=0.005..0.005 rows=1 loops=7997)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=31988
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.169..0.171 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.043..0.157 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.029..0.133 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.015..0.015 rows=53 loops=1)
                                            Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.025 ms
Execution Time: 68.622 ms
```

### text in (2 values) 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1508.64..1508.65 rows=1 width=8) (actual time=17.470..17.486 rows=1 loops=1)
  Buffers: shared hit=16592
  ->  Limit  (cost=105.97..1508.50 rows=11 width=4) (actual time=0.930..17.448 rows=72 loops=1)
        Buffers: shared hit=16592
        ->  Result  (cost=105.97..1508.50 rows=11 width=4) (actual time=0.929..17.429 rows=72 loops=1)
              Buffers: shared hit=16592
              ->  Append  (cost=105.97..1508.39 rows=11 width=0) (actual time=0.927..17.403 rows=72 loops=1)
                    Buffers: shared hit=16592
                    ->  Subquery Scan on "*SELECT* 1"  (cost=105.97..637.25 rows=8 width=0) (actual time=0.926..14.761 rows=71 loops=1)
                          Buffers: shared hit=15702
                          ->  Limit  (cost=105.97..637.17 rows=8 width=4) (actual time=0.925..14.734 rows=71 loops=1)
                                Buffers: shared hit=15702
                                ->  Result  (cost=105.97..637.17 rows=8 width=4) (actual time=0.924..14.715 rows=71 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15702
                                      ->  Hash Semi Join  (cost=105.97..637.17 rows=8 width=0) (actual time=0.911..14.679 rows=71 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15702
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.044..9.735 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=91.02..91.02 rows=1151 width=32) (actual time=0.735..0.736 rows=962 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                  Buffers: shared hit=21
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.034..0.434 rows=962 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=21
                    ->  Subquery Scan on "*SELECT* 2"  (cost=119.96..667.16 rows=1 width=0) (actual time=1.611..1.788 rows=1 loops=1)
                          Buffers: shared hit=542
                          ->  Limit  (cost=119.96..667.15 rows=1 width=4) (actual time=1.609..1.786 rows=1 loops=1)
                                Buffers: shared hit=542
                                ->  Result  (cost=119.96..667.15 rows=1 width=4) (actual time=1.607..1.783 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=542
                                      ->  Hash Semi Join  (cost=119.96..667.15 rows=1 width=0) (actual time=1.592..1.767 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=542
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.250..1.185 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.121..0.121 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=91.02..91.02 rows=1151 width=32) (actual time=0.482..0.483 rows=962 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                  Buffers: shared hit=21
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.027..0.279 rows=962 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=21
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.17 rows=1 width=0) (actual time=0.696..0.698 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..158.16 rows=1 width=4) (actual time=0.695..0.697 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=4) (actual time=0.694..0.695 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=0) (actual time=0.685..0.686 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.026..0.173 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p025a,p025b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.76 rows=1 width=0) (actual time=0.122..0.125 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.75 rows=1 width=4) (actual time=0.120..0.124 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.75 rows=1 width=4) (actual time=0.119..0.122 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=0) (actual time=0.113..0.116 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.066..0.085 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.051 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.040 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.026..0.027 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p025a,p025b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.869 ms
Execution Time: 17.717 ms
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
Nested Loop Left Join  (cost=10156.84..10330.64 rows=1 width=1434) (actual time=9.670..10.057 rows=51 loops=1)
  Buffers: shared hit=5065
  ->  Nested Loop  (cost=10156.54..10322.30 rows=1 width=1399) (actual time=9.650..9.878 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4912
        ->  Subquery Scan on x  (cost=10156.26..10313.99 rows=1 width=1392) (actual time=9.636..9.740 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4759
              ->  Limit  (cost=10156.26..10313.74 rows=11 width=1392) (actual time=9.607..9.666 rows=51 loops=1)
                    Buffers: shared hit=4759
                    ->  Merge Append  (cost=10156.26..10313.74 rows=11 width=1392) (actual time=9.605..9.658 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4759
                          ->  Limit  (cost=9442.43..9442.45 rows=8 width=1392) (actual time=7.180..7.200 rows=51 loops=1)
                                Buffers: shared hit=3869
                                ->  Sort  (cost=9442.43..9442.45 rows=8 width=1392) (actual time=7.178..7.193 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 167kB
                                      Buffers: shared hit=3869
                                      ->  Result  (cost=97.21..9442.31 rows=8 width=1392) (actual time=0.621..7.090 rows=71 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=3869
                                            ->  Nested Loop  (cost=97.21..9442.31 rows=8 width=1384) (actual time=0.615..7.055 rows=71 loops=1)
                                                  Buffers: shared hit=3869
                                                  ->  HashAggregate  (cost=96.77..108.28 rows=1151 width=32) (actual time=0.557..0.794 rows=962 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=21
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.038..0.248 rows=962 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=21
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.11 rows=1 width=1384) (actual time=0.006..0.006 rows=0 loops=962)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=3848
                          ->  Limit  (cost=667.16..667.16 rows=1 width=1392) (actual time=1.715..1.719 rows=1 loops=1)
                                Buffers: shared hit=542
                                ->  Sort  (cost=667.16..667.16 rows=1 width=1392) (actual time=1.713..1.716 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=542
                                      ->  Result  (cost=119.96..667.15 rows=1 width=1392) (actual time=1.529..1.703 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=542
                                            ->  Hash Semi Join  (cost=119.96..667.15 rows=1 width=1384) (actual time=1.515..1.688 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=542
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.208..1.130 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.109..0.109 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=91.02..91.02 rows=1151 width=32) (actual time=0.480..0.481 rows=962 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 77kB
                                                        Buffers: shared hit=21
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.026..0.262 rows=962 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=21
                          ->  Limit  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.595..0.597 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.594..0.595 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=1384) (actual time=0.587..0.588 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.020..0.129 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p025a,p025b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.108..0.112 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.107..0.111 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.75 rows=1 width=1392) (actual time=0.101..0.104 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=1384) (actual time=0.095..0.098 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.053..0.072 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.039..0.042 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.034 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                        Filter: (value = ANY ('{p025a,p025b}'::text[]))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.064 ms
Execution Time: 10.301 ms
```

### text in (2 values) 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=9435.24..9435.25 rows=1 width=8) (actual time=10.156..10.161 rows=1 loops=1)
  Buffers: shared hit=3869
  ->  Limit  (cost=97.21..9431.68 rows=285 width=4) (actual time=0.602..9.944 rows=962 loops=1)
        Buffers: shared hit=3869
        ->  Result  (cost=97.21..9431.68 rows=285 width=4) (actual time=0.601..9.717 rows=962 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=3869
              ->  Nested Loop  (cost=97.21..9431.68 rows=285 width=0) (actual time=0.594..9.427 rows=962 loops=1)
                    Buffers: shared hit=3869
                    ->  HashAggregate  (cost=96.77..108.28 rows=1151 width=32) (actual time=0.574..0.971 rows=962 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=21
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.033..0.219 rows=962 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                Heap Fetches: 0
                                Buffers: shared hit=21
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.10 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=962)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=3848
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.500 ms
Execution Time: 10.230 ms
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
Limit  (cost=241.17..241.30 rows=51 width=1426) (actual time=16.553..16.572 rows=51 loops=1)
  Buffers: shared hit=6758
  ->  Sort  (cost=241.17..241.88 rows=285 width=1426) (actual time=16.551..16.564 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 207kB
        Buffers: shared hit=6758
        ->  Result  (cost=97.77..231.66 rows=285 width=1426) (actual time=1.113..15.358 rows=962 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=6758
              ->  Nested Loop Left Join  (cost=97.77..229.53 rows=285 width=1399) (actual time=1.089..14.436 rows=962 loops=1)
                    Buffers: shared hit=6758
                    ->  Nested Loop  (cost=97.47..136.27 rows=285 width=1391) (actual time=1.066..9.960 rows=962 loops=1)
                          Buffers: shared hit=3872
                          ->  Nested Loop  (cost=97.04..128.08 rows=1 width=71) (actual time=1.034..1.675 rows=962 loops=1)
                                Buffers: shared hit=24
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.036..0.039 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=96.77..108.28 rows=1151 width=32) (actual time=0.996..1.338 rows=962 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=21
                                      ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.069..0.446 rows=962 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p025a,p025b}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=21
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.10 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=962)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3848
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=962)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=2886
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=962)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=2886
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.718 ms
Execution Time: 16.818 ms
```

### text in (2 values) 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = ANY($9::text[])) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1508.64..1508.65 rows=1 width=8) (actual time=13.929..13.946 rows=1 loops=1)
  Buffers: shared hit=16568
  ->  Limit  (cost=105.97..1508.50 rows=11 width=4) (actual time=13.926..13.942 rows=0 loops=1)
        Buffers: shared hit=16568
        ->  Result  (cost=105.97..1508.50 rows=11 width=4) (actual time=13.924..13.940 rows=0 loops=1)
              Buffers: shared hit=16568
              ->  Append  (cost=105.97..1508.39 rows=11 width=0) (actual time=13.923..13.939 rows=0 loops=1)
                    Buffers: shared hit=16568
                    ->  Subquery Scan on "*SELECT* 1"  (cost=105.97..637.25 rows=8 width=0) (actual time=11.336..11.341 rows=0 loops=1)
                          Buffers: shared hit=15690
                          ->  Limit  (cost=105.97..637.17 rows=8 width=4) (actual time=11.334..11.339 rows=0 loops=1)
                                Buffers: shared hit=15690
                                ->  Result  (cost=105.97..637.17 rows=8 width=4) (actual time=11.333..11.337 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15690
                                      ->  Hash Semi Join  (cost=105.97..637.17 rows=8 width=0) (actual time=11.321..11.325 rows=0 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15690
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.033..8.214 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=91.02..91.02 rows=1151 width=32) (actual time=0.053..0.054 rows=43 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                  Buffers: shared hit=9
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.026..0.041 rows=43 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=9
                    ->  Subquery Scan on "*SELECT* 2"  (cost=119.96..667.16 rows=1 width=0) (actual time=1.752..1.756 rows=0 loops=1)
                          Buffers: shared hit=530
                          ->  Limit  (cost=119.96..667.15 rows=1 width=4) (actual time=1.750..1.753 rows=0 loops=1)
                                Buffers: shared hit=530
                                ->  Result  (cost=119.96..667.15 rows=1 width=4) (actual time=1.746..1.749 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=530
                                      ->  Hash Semi Join  (cost=119.96..667.15 rows=1 width=0) (actual time=1.726..1.729 rows=0 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=530
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.256..1.391 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.137..0.137 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=91.02..91.02 rows=1151 width=32) (actual time=0.239..0.240 rows=43 loops=1)
                                                  Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                  Buffers: shared hit=9
                                                  ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.179..0.214 rows=43 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=9
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.17 rows=1 width=0) (actual time=0.702..0.703 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..158.16 rows=1 width=4) (actual time=0.700..0.701 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=4) (actual time=0.698..0.699 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=0) (actual time=0.683..0.683 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.035..0.152 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p001a,p001b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.76 rows=1 width=0) (actual time=0.126..0.129 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.75 rows=1 width=4) (actual time=0.125..0.128 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.75 rows=1 width=4) (actual time=0.124..0.126 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=0) (actual time=0.116..0.118 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.071..0.090 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.054..0.057 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.038..0.043 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p001a,p001b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.558 ms
Execution Time: 14.110 ms
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
Nested Loop Left Join  (cost=10156.84..10330.64 rows=1 width=1434) (actual time=2.671..2.688 rows=0 loops=1)
  Buffers: shared hit=1059
  ->  Nested Loop  (cost=10156.54..10322.30 rows=1 width=1399) (actual time=2.671..2.687 rows=0 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=1059
        ->  Subquery Scan on x  (cost=10156.26..10313.99 rows=1 width=1392) (actual time=2.670..2.685 rows=0 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=1059
              ->  Limit  (cost=10156.26..10313.74 rows=11 width=1392) (actual time=2.669..2.683 rows=0 loops=1)
                    Buffers: shared hit=1059
                    ->  Merge Append  (cost=10156.26..10313.74 rows=11 width=1392) (actual time=2.667..2.682 rows=0 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=1059
                          ->  Limit  (cost=9442.43..9442.45 rows=8 width=1392) (actual time=0.480..0.485 rows=0 loops=1)
                                Buffers: shared hit=181
                                ->  Sort  (cost=9442.43..9442.45 rows=8 width=1392) (actual time=0.479..0.482 rows=0 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=181
                                      ->  Result  (cost=97.21..9442.31 rows=8 width=1392) (actual time=0.475..0.478 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=181
                                            ->  Nested Loop  (cost=97.21..9442.31 rows=8 width=1384) (actual time=0.471..0.473 rows=0 loops=1)
                                                  Buffers: shared hit=181
                                                  ->  HashAggregate  (cost=96.77..108.28 rows=1151 width=32) (actual time=0.074..0.094 rows=43 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 73kB
                                                        Buffers: shared hit=9
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.042..0.054 rows=43 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=9
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.11 rows=1 width=1384) (actual time=0.008..0.008 rows=0 loops=43)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=172
                          ->  Limit  (cost=667.16..667.16 rows=1 width=1392) (actual time=1.410..1.414 rows=0 loops=1)
                                Buffers: shared hit=530
                                ->  Sort  (cost=667.16..667.16 rows=1 width=1392) (actual time=1.409..1.413 rows=0 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=530
                                      ->  Result  (cost=119.96..667.15 rows=1 width=1392) (actual time=1.405..1.408 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=530
                                            ->  Hash Semi Join  (cost=119.96..667.15 rows=1 width=1384) (actual time=1.402..1.404 rows=0 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=530
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.216..1.299 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.119..0.119 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=91.02..91.02 rows=1151 width=32) (actual time=0.037..0.037 rows=43 loops=1)
                                                        Buckets: 2048  Batches: 1  Memory Usage: 19kB
                                                        Buffers: shared hit=9
                                                        ->  Index Only Scan using rfv_text_value on rfv_text v_1  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.013..0.026 rows=43 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=9
                          ->  Limit  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.662..0.663 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.660..0.661 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=1384) (actual time=0.652..0.653 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.024..0.146 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_text_pkey on rfv_text v_2  (cost=0.43..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                  Filter: (value = ANY ('{p001a,p001b}'::text[]))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.113..0.116 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.76..45.77 rows=1 width=1392) (actual time=0.112..0.115 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.75 rows=1 width=1392) (actual time=0.105..0.108 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.75 rows=1 width=1384) (actual time=0.101..0.104 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.057..0.076 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.042..0.046 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.032..0.039 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.023..0.025 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_text_pkey on rfv_text v_3  (cost=0.43..1.28 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid))
                                                        Filter: (value = ANY ('{p001a,p001b}'::text[]))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (never executed)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (never executed)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.373 ms
Execution Time: 2.923 ms
```

### text in (2 values) 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = ANY($8::text[])) LIMIT 1001) c
```

```
Aggregate  (cost=9435.24..9435.25 rows=1 width=8) (actual time=0.477..0.480 rows=1 loops=1)
  Buffers: shared hit=181
  ->  Limit  (cost=97.21..9431.68 rows=285 width=4) (actual time=0.097..0.471 rows=43 loops=1)
        Buffers: shared hit=181
        ->  Result  (cost=97.21..9431.68 rows=285 width=4) (actual time=0.096..0.463 rows=43 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=181
              ->  Nested Loop  (cost=97.21..9431.68 rows=285 width=0) (actual time=0.089..0.449 rows=43 loops=1)
                    Buffers: shared hit=181
                    ->  HashAggregate  (cost=96.77..108.28 rows=1151 width=32) (actual time=0.070..0.089 rows=43 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 73kB
                          Buffers: shared hit=9
                          ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.041..0.054 rows=43 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                Heap Fetches: 0
                                Buffers: shared hit=9
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.10 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=43)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=172
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.581 ms
Execution Time: 0.539 ms
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
Limit  (cost=241.17..241.30 rows=51 width=1426) (actual time=0.811..0.823 rows=43 loops=1)
  Buffers: shared hit=313
  ->  Sort  (cost=241.17..241.88 rows=285 width=1426) (actual time=0.810..0.817 rows=43 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 111kB
        Buffers: shared hit=313
        ->  Result  (cost=97.77..231.66 rows=285 width=1426) (actual time=0.122..0.759 rows=43 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=313
              ->  Nested Loop Left Join  (cost=97.77..229.53 rows=285 width=1399) (actual time=0.109..0.716 rows=43 loops=1)
                    Buffers: shared hit=313
                    ->  Nested Loop  (cost=97.47..136.27 rows=285 width=1391) (actual time=0.095..0.524 rows=43 loops=1)
                          Buffers: shared hit=184
                          ->  Nested Loop  (cost=97.04..128.08 rows=1 width=71) (actual time=0.077..0.146 rows=43 loops=1)
                                Buffers: shared hit=12
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.022..0.024 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=96.77..108.28 rows=1151 width=32) (actual time=0.054..0.112 rows=43 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 73kB
                                      Buffers: shared hit=9
                                      ->  Index Only Scan using rfv_text_value on rfv_text v  (cost=0.55..91.02 rows=1151 width=32) (actual time=0.025..0.037 rows=43 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '5408e1d0-8310-4c18-92fa-d063a02e97b6'::uuid) AND (value = ANY ('{p001a,p001b}'::text[])))
                                            Heap Fetches: 0
                                            Buffers: shared hit=9
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.10 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=43)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=172
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=43)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=129
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=43)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=129
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.954 ms
Execution Time: 0.914 ms
```

### num lt 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=17739.06..17739.07 rows=1 width=8) (actual time=86.414..86.424 rows=1 loops=1)
  Buffers: shared hit=5129, temp written=523
  I/O Timings: temp write=4.940
  ->  Limit  (cost=14408.49..17726.55 rows=1001 width=4) (actual time=81.930..86.303 rows=1001 loops=1)
        Buffers: shared hit=5129, temp written=523
        I/O Timings: temp write=4.940
        ->  Result  (cost=14408.49..17825.99 rows=1031 width=4) (actual time=81.929..86.175 rows=1001 loops=1)
              Buffers: shared hit=5129, temp written=523
              I/O Timings: temp write=4.940
              ->  Append  (cost=14408.49..17815.68 rows=1031 width=0) (actual time=81.928..86.005 rows=1001 loops=1)
                    Buffers: shared hit=5129, temp written=523
                    I/O Timings: temp write=4.940
                    ->  Subquery Scan on "*SELECT* 1"  (cost=14408.49..15894.92 rows=1001 width=0) (actual time=81.926..85.866 rows=1001 loops=1)
                          Buffers: shared hit=5129, temp written=523
                          I/O Timings: temp write=4.940
                          ->  Limit  (cost=14408.49..15884.91 rows=1001 width=4) (actual time=81.926..85.728 rows=1001 loops=1)
                                Buffers: shared hit=5129, temp written=523
                                I/O Timings: temp write=4.940
                                ->  Result  (cost=14408.49..16116.47 rows=1158 width=4) (actual time=81.924..85.620 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=5129, temp written=523
                                      I/O Timings: temp write=4.940
                                      ->  Hash Semi Join  (cost=14408.49..16116.47 rows=1158 width=0) (actual time=81.916..85.432 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=5129, temp written=523
                                            I/O Timings: temp write=4.940
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.050..1.722 rows=4910 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=3195
                                            ->  Hash  (cost=11342.67..11342.67 rows=158500 width=32) (actual time=81.770..81.771 rows=160258 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 6033kB
                                                  Buffers: shared hit=1934, temp written=508
                                                  I/O Timings: temp write=4.782
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..11342.67 rows=158500 width=32) (actual time=0.022..36.586 rows=160258 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '4000'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1934
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1712.10 rows=27 width=0) (never executed)
                          ->  Limit  (cost=0.86..1711.83 rows=27 width=4) (never executed)
                                ->  Result  (cost=0.86..1711.83 rows=27 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1711.83 rows=27 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_1.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '4000'::double precision)
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.15 rows=2 width=0) (never executed)
                          ->  Limit  (cost=0.86..158.13 rows=2 width=4) (never executed)
                                ->  Result  (cost=0.86..158.13 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..158.13 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '4000'::double precision)
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '4000'::double precision)
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.780 ms
Execution Time: 87.547 ms
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
Nested Loop Left Join  (cost=48.57..16218.46 rows=1 width=1434) (actual time=0.462..3.659 rows=51 loops=1)
  Buffers: shared hit=2397
  ->  Nested Loop  (cost=48.27..16210.12 rows=1 width=1399) (actual time=0.446..3.467 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2244
        ->  Subquery Scan on x  (cost=47.99..16201.81 rows=1 width=1392) (actual time=0.437..3.330 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2091
              ->  Limit  (cost=47.99..16200.66 rows=51 width=1392) (actual time=0.435..3.272 rows=51 loops=1)
                    Buffers: shared hit=2091
                    ->  Merge Append  (cost=47.99..25702.23 rows=81 width=1392) (actual time=0.433..3.261 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2091
                          ->  Limit  (cost=0.86..23785.65 rows=51 width=1392) (actual time=0.187..2.990 rows=51 loops=1)
                                Buffers: shared hit=2013
                                ->  Result  (cost=0.86..540055.41 rows=1158 width=1392) (actual time=0.186..2.980 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=2013
                                      ->  Nested Loop Semi Join  (cost=0.86..540055.41 rows=1158 width=1384) (actual time=0.181..2.956 rows=51 loops=1)
                                            Buffers: shared hit=2013
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..501702.69 rows=5845 width=1384) (actual time=0.081..2.316 rows=102 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1482
                                                  Buffers: shared hit=1605
                                            ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..6.55 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=102)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '4000'::double precision)
                                                  Rows Removed by Filter: 0
                                                  Buffers: shared hit=408
                          ->  Limit  (cost=0.86..1711.83 rows=27 width=1392) (actual time=0.048..0.051 rows=1 loops=1)
                                Buffers: shared hit=18
                                ->  Result  (cost=0.86..1711.83 rows=27 width=1392) (actual time=0.047..0.049 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=18
                                      ->  Nested Loop Semi Join  (cost=0.86..1711.83 rows=27 width=1384) (actual time=0.044..0.046 rows=1 loops=1)
                                            Buffers: shared hit=18
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.012..0.017 rows=3 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=6
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_1  (cost=0.42..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_1.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '4000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
                          ->  Limit  (cost=0.86..158.13 rows=2 width=1392) (actual time=0.087..0.089 rows=1 loops=1)
                                Buffers: shared hit=23
                                ->  Result  (cost=0.86..158.13 rows=2 width=1392) (actual time=0.087..0.089 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=23
                                      ->  Nested Loop Semi Join  (cost=0.86..158.13 rows=2 width=1384) (actual time=0.085..0.087 rows=1 loops=1)
                                            Buffers: shared hit=23
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.015..0.024 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=7
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.015..0.015 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '4000'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=16
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.109..0.114 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.108..0.113 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 31kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1392) (actual time=0.062..0.103 rows=3 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1384) (actual time=0.059..0.099 rows=3 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.050..0.072 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.041 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.027..0.033 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.020..0.022 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.008 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                        Filter: (value < '4000'::double precision)
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.525 ms
Execution Time: 3.863 ms
```

### num lt 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=4604.87..4604.88 rows=1 width=8) (actual time=16.222..16.225 rows=1 loops=1)
  Buffers: shared hit=11821
  ->  Limit  (cost=0.86..4592.36 rows=1001 width=4) (actual time=0.101..16.024 rows=1001 loops=1)
        Buffers: shared hit=11821
        ->  Result  (cost=0.86..180041.72 rows=39251 width=4) (actual time=0.100..15.819 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=11821
              ->  Nested Loop Semi Join  (cost=0.86..180041.72 rows=39251 width=0) (actual time=0.087..15.569 rows=1001 loops=1)
                    Buffers: shared hit=11821
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.042..1.141 rows=2541 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1657
                    ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..0.83 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2541)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                          Filter: (value < '4000'::double precision)
                          Rows Removed by Filter: 1
                          Buffers: shared hit=10164
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.820 ms
Execution Time: 16.288 ms
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
Limit  (cost=1.42..883.74 rows=51 width=1426) (actual time=0.119..1.463 rows=51 loops=1)
  Buffers: shared hit=801
  ->  Result  (cost=1.42..679056.86 rows=39251 width=1426) (actual time=0.117..1.451 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
        Buffers: shared hit=801
        ->  Nested Loop Left Join  (cost=1.42..678762.48 rows=39251 width=1399) (actual time=0.098..1.397 rows=51 loops=1)
              Buffers: shared hit=801
              ->  Nested Loop  (cost=1.12..665921.53 rows=39251 width=1391) (actual time=0.077..1.167 rows=51 loops=1)
                    Buffers: shared hit=648
                    ->  Nested Loop Semi Join  (cost=0.85..665422.60 rows=39251 width=1384) (actual time=0.063..1.119 rows=51 loops=1)
                          Buffers: shared hit=645
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..500712.14 rows=198111 width=1384) (actual time=0.030..0.287 rows=128 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                Buffers: shared hit=133
                          ->  Index Scan using rfv_num_pkey on rfv_num v  (cost=0.42..0.83 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=128)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                Filter: (value < '4000'::double precision)
                                Rows Removed by Filter: 1
                                Buffers: shared hit=512
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.000 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.009..0.010 rows=1 loops=1)
                                Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.456 ms
Execution Time: 1.631 ms
```

### num lt 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2615.62..2615.63 rows=1 width=8) (actual time=20.380..20.392 rows=1 loops=1)
  Buffers: shared hit=16766
  ->  Limit  (cost=658.48..2614.87 rows=60 width=4) (actual time=2.978..20.310 rows=487 loops=1)
        Buffers: shared hit=16766
        ->  Result  (cost=658.48..2614.87 rows=60 width=4) (actual time=2.977..20.231 rows=487 loops=1)
              Buffers: shared hit=16766
              ->  Append  (cost=658.48..2614.27 rows=60 width=0) (actual time=2.977..20.126 rows=487 loops=1)
                    Buffers: shared hit=16766
                    ->  Subquery Scan on "*SELECT* 1"  (cost=658.48..1190.78 rows=57 width=0) (actual time=2.976..15.437 rows=481 loops=1)
                          Buffers: shared hit=15789
                          ->  Limit  (cost=658.48..1190.21 rows=57 width=4) (actual time=2.975..15.350 rows=481 loops=1)
                                Buffers: shared hit=15789
                                ->  Result  (cost=658.48..1190.21 rows=57 width=4) (actual time=2.974..15.277 rows=481 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15789
                                      ->  Hash Semi Join  (cost=658.48..1190.21 rows=57 width=0) (actual time=2.969..15.179 rows=481 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15789
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.024..7.750 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=560.24..560.24 rows=7814 width=32) (actual time=2.933..2.934 rows=8098 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                  Buffers: shared hit=108
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..560.24 rows=7814 width=32) (actual time=0.017..1.638 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=108
                    ->  Subquery Scan on "*SELECT* 2"  (cost=672.47..1219.68 rows=1 width=0) (actual time=3.593..4.022 rows=4 loops=1)
                          Buffers: shared hit=629
                          ->  Limit  (cost=672.47..1219.67 rows=1 width=4) (actual time=3.592..4.020 rows=4 loops=1)
                                Buffers: shared hit=629
                                ->  Result  (cost=672.47..1219.67 rows=1 width=4) (actual time=3.590..4.017 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=629
                                      ->  Hash Semi Join  (cost=672.47..1219.67 rows=1 width=0) (actual time=3.578..4.004 rows=4 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=629
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.188..0.987 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.103..0.103 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=560.24..560.24 rows=7814 width=32) (actual time=2.902..2.902 rows=8098 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                  Buffers: shared hit=108
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..560.24 rows=7814 width=32) (actual time=0.019..1.601 rows=8098 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '200'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=108
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.14 rows=1 width=0) (actual time=0.312..0.492 rows=2 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..158.13 rows=1 width=4) (actual time=0.311..0.491 rows=2 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.13 rows=1 width=4) (actual time=0.310..0.489 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.13 rows=1 width=0) (actual time=0.302..0.480 rows=2 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.019..0.106 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (actual time=0.087..0.089 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (actual time=0.086..0.088 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (actual time=0.085..0.088 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (actual time=0.083..0.085 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.049..0.064 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.038..0.040 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.027..0.032 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.006 ms
Execution Time: 20.530 ms
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
Nested Loop Left Join  (cost=22219.88..22386.52 rows=1 width=1434) (actual time=48.761..49.050 rows=51 loops=1)
  Buffers: shared hit=23061
  ->  Nested Loop  (cost=22219.58..22378.18 rows=1 width=1399) (actual time=48.745..48.916 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22908
        ->  Subquery Scan on x  (cost=22219.30..22369.87 rows=1 width=1392) (actual time=48.736..48.815 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22755
              ->  Limit  (cost=22219.30..22368.72 rows=51 width=1392) (actual time=48.733..48.781 rows=51 loops=1)
                    Buffers: shared hit=22755
                    ->  Merge Append  (cost=22219.30..22377.51 rows=54 width=1392) (actual time=48.731..48.774 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22755
                          ->  Limit  (cost=20953.35..20953.48 rows=51 width=1392) (actual time=44.764..44.784 rows=51 loops=1)
                                Buffers: shared hit=21905
                                ->  Sort  (cost=20953.35..20953.49 rows=57 width=1392) (actual time=44.762..44.777 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 207kB
                                      Buffers: shared hit=21905
                                      ->  Result  (cost=1117.25..20951.69 rows=57 width=1392) (actual time=11.087..44.341 rows=481 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=21905
                                            ->  Hash Semi Join  (cost=1117.25..20951.69 rows=57 width=1384) (actual time=11.083..44.202 rows=481 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21905
                                                  ->  Bitmap Heap Scan on records r  (cost=459.32..20277.78 rows=5845 width=1384) (actual time=8.173..37.306 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.86 rows=5845 width=0) (actual time=3.823..3.823 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=560.24..560.24 rows=7814 width=32) (actual time=2.722..2.723 rows=8098 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                        Buffers: shared hit=108
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..560.24 rows=7814 width=32) (actual time=0.021..1.479 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=108
                          ->  Limit  (cost=1219.68..1219.69 rows=1 width=1392) (actual time=3.552..3.554 rows=1 loops=1)
                                Buffers: shared hit=629
                                ->  Sort  (cost=1219.68..1219.69 rows=1 width=1392) (actual time=3.549..3.551 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 33kB
                                      Buffers: shared hit=629
                                      ->  Result  (cost=672.47..1219.67 rows=1 width=1392) (actual time=3.162..3.535 rows=4 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=629
                                            ->  Hash Semi Join  (cost=672.47..1219.67 rows=1 width=1384) (actual time=3.151..3.522 rows=4 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=629
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.157..0.839 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.085..0.086 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=560.24..560.24 rows=7814 width=32) (actual time=2.591..2.591 rows=8098 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 571kB
                                                        Buffers: shared hit=108
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..560.24 rows=7814 width=32) (actual time=0.016..1.433 rows=8098 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '200'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=108
                          ->  Limit  (cost=0.86..158.13 rows=1 width=1392) (actual time=0.275..0.277 rows=1 loops=1)
                                Buffers: shared hit=184
                                ->  Result  (cost=0.86..158.13 rows=1 width=1392) (actual time=0.274..0.275 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=184
                                      ->  Nested Loop Semi Join  (cost=0.86..158.13 rows=1 width=1384) (actual time=0.266..0.267 rows=1 loops=1)
                                            Buffers: shared hit=184
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.018..0.064 rows=36 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=40
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=36)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '200'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=144
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.135..0.138 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.134..0.137 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1392) (actual time=0.109..0.111 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1384) (actual time=0.105..0.107 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.074..0.087 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.061..0.063 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.049..0.053 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.042..0.043 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.010..0.010 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                        Filter: (value < '200'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.266 ms
Execution Time: 49.241 ms
```

### num lt 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=8060.05..8060.06 rows=1 width=8) (actual time=25.530..25.534 rows=1 loops=1)
  Buffers: shared hit=32372
  ->  Limit  (cost=658.35..8047.54 rows=1001 width=4) (actual time=4.021..25.412 rows=1001 loops=1)
        Buffers: shared hit=32372
        ->  Result  (cost=658.35..14942.14 rows=1935 width=4) (actual time=4.018..25.278 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32372
              ->  Hash Semi Join  (cost=658.35..14942.14 rows=1935 width=0) (actual time=4.010..25.111 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=32372
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.026..13.428 rows=49778 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=32264
                    ->  Hash  (cost=560.24..560.24 rows=7814 width=32) (actual time=3.951..3.952 rows=8098 loops=1)
                          Buckets: 8192  Batches: 1  Memory Usage: 571kB
                          Buffers: shared hit=108
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..560.24 rows=7814 width=32) (actual time=0.018..2.177 rows=8098 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '200'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=108
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.628 ms
Execution Time: 25.580 ms
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
Limit  (cost=1000.05..1000.18 rows=51 width=1426) (actual time=66.899..66.919 rows=51 loops=1)
  Buffers: shared hit=32557
  ->  Sort  (cost=1000.05..1004.89 rows=1935 width=1426) (actual time=66.897..66.912 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 205kB
        Buffers: shared hit=32557
        ->  Result  (cost=744.71..935.50 rows=1935 width=1426) (actual time=3.906..61.103 rows=8098 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32557
              ->  Hash Left Join  (cost=744.71..920.98 rows=1935 width=1399) (actual time=3.893..56.430 rows=8098 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32557
                    ->  Nested Loop  (cost=600.01..771.20 rows=1935 width=1391) (actual time=3.694..52.983 rows=8098 loops=1)
                          Buffers: shared hit=32503
                          ->  Nested Loop  (cost=599.58..763.88 rows=1 width=71) (actual time=3.672..7.574 rows=8098 loops=1)
                                Buffers: shared hit=111
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.018..0.022 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=599.31..677.45 rows=7814 width=32) (actual time=3.652..6.080 rows=8098 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=108
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..560.24 rows=7814 width=32) (actual time=0.020..1.377 rows=8098 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '200'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=108
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.83 rows=1 width=1384) (actual time=0.005..0.005 rows=1 loops=8098)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=32392
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.191..0.193 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.029..0.180 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.024..0.166 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.011..0.011 rows=53 loops=1)
                                            Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.817 ms
Execution Time: 67.042 ms
```

### num lt 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1441.20..1441.21 rows=1 width=8) (actual time=13.097..13.108 rows=1 loops=1)
  Buffers: shared hit=16357
  ->  Limit  (cost=82.34..1441.07 rows=10 width=4) (actual time=0.877..13.095 rows=40 loops=1)
        Buffers: shared hit=16357
        ->  Result  (cost=82.34..1441.07 rows=10 width=4) (actual time=0.876..13.086 rows=40 loops=1)
              Buffers: shared hit=16357
              ->  Append  (cost=82.34..1440.97 rows=10 width=0) (actual time=0.876..13.075 rows=40 loops=1)
                    Buffers: shared hit=16357
                    ->  Subquery Scan on "*SELECT* 1"  (cost=82.34..613.59 rows=7 width=0) (actual time=0.875..10.832 rows=39 loops=1)
                          Buffers: shared hit=15698
                          ->  Limit  (cost=82.34..613.52 rows=7 width=4) (actual time=0.874..10.822 rows=39 loops=1)
                                Buffers: shared hit=15698
                                ->  Result  (cost=82.34..613.52 rows=7 width=4) (actual time=0.873..10.814 rows=39 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15698
                                      ->  Hash Semi Join  (cost=82.34..613.52 rows=7 width=0) (actual time=0.866..10.797 rows=39 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15698
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.028..6.974 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=69.87..69.87 rows=953 width=32) (actual time=0.569..0.570 rows=1002 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 71kB
                                                  Buffers: shared hit=17
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..69.87 rows=953 width=32) (actual time=0.023..0.320 rows=1002 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=17
                    ->  Subquery Scan on "*SELECT* 2"  (cost=96.34..643.54 rows=1 width=0) (actual time=1.173..1.526 rows=1 loops=1)
                          Buffers: shared hit=538
                          ->  Limit  (cost=96.34..643.53 rows=1 width=4) (actual time=1.171..1.524 rows=1 loops=1)
                                Buffers: shared hit=538
                                ->  Result  (cost=96.34..643.53 rows=1 width=4) (actual time=1.170..1.522 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=538
                                      ->  Hash Semi Join  (cost=96.34..643.53 rows=1 width=0) (actual time=1.162..1.513 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=538
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.210..0.982 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.103..0.103 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=69.87..69.87 rows=953 width=32) (actual time=0.453..0.453 rows=1002 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 71kB
                                                  Buffers: shared hit=17
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..69.87 rows=953 width=32) (actual time=0.019..0.272 rows=1002 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=17
                    ->  Subquery Scan on "*SELECT* 3"  (cost=82.22..138.43 rows=1 width=0) (actual time=0.514..0.515 rows=0 loops=1)
                          Buffers: shared hit=84
                          ->  Limit  (cost=82.22..138.42 rows=1 width=4) (actual time=0.513..0.514 rows=0 loops=1)
                                Buffers: shared hit=84
                                ->  Result  (cost=82.22..138.42 rows=1 width=4) (actual time=0.512..0.513 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=84
                                      ->  Hash Semi Join  (cost=82.22..138.42 rows=1 width=0) (actual time=0.507..0.507 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=84
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.018..0.104 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=69.87..69.87 rows=953 width=32) (actual time=0.391..0.391 rows=1002 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 71kB
                                                  Buffers: shared hit=17
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..69.87 rows=953 width=32) (actual time=0.012..0.213 rows=1002 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=17
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (actual time=0.186..0.189 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (actual time=0.185..0.188 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (actual time=0.184..0.186 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (actual time=0.182..0.184 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.135..0.151 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.119..0.121 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.091..0.097 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.083..0.084 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.070..0.070 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '25'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.070 ms
Execution Time: 13.252 ms
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
Nested Loop Left Join  (cost=8699.41..8715.88 rows=1 width=1434) (actual time=8.532..8.798 rows=40 loops=1)
  Buffers: shared hit=4924
  ->  Nested Loop  (cost=8699.11..8707.55 rows=1 width=1399) (actual time=8.516..8.672 rows=40 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4804
        ->  Subquery Scan on x  (cost=8698.83..8699.24 rows=1 width=1392) (actual time=8.506..8.576 rows=40 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4684
              ->  Limit  (cost=8698.83..8699.01 rows=10 width=1392) (actual time=8.503..8.543 rows=40 loops=1)
                    Buffers: shared hit=4684
                    ->  Merge Append  (cost=8698.83..8699.01 rows=10 width=1392) (actual time=8.502..8.537 rows=40 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4684
                          ->  Limit  (cost=7871.46..7871.47 rows=7 width=1392) (actual time=6.445..6.459 rows=39 loops=1)
                                Buffers: shared hit=4025
                                ->  Sort  (cost=7871.46..7871.47 rows=7 width=1392) (actual time=6.444..6.454 rows=39 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 103kB
                                      Buffers: shared hit=4025
                                      ->  Result  (cost=75.07..7871.36 rows=7 width=1392) (actual time=0.613..6.396 rows=39 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=4025
                                            ->  Nested Loop  (cost=75.07..7871.36 rows=7 width=1384) (actual time=0.608..6.376 rows=39 loops=1)
                                                  Buffers: shared hit=4025
                                                  ->  HashAggregate  (cost=74.63..84.16 rows=953 width=32) (actual time=0.519..0.759 rows=1002 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=17
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..69.87 rows=953 width=32) (actual time=0.024..0.192 rows=1002 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=17
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.17 rows=1 width=1384) (actual time=0.005..0.005 rows=0 loops=1002)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=4008
                          ->  Limit  (cost=643.54..643.54 rows=1 width=1392) (actual time=1.433..1.436 rows=1 loops=1)
                                Buffers: shared hit=538
                                ->  Sort  (cost=643.54..643.54 rows=1 width=1392) (actual time=1.431..1.434 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=538
                                      ->  Result  (cost=96.34..643.53 rows=1 width=1392) (actual time=1.061..1.423 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=538
                                            ->  Hash Semi Join  (cost=96.34..643.53 rows=1 width=1384) (actual time=1.053..1.415 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=538
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.182..0.956 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.097..0.097 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=69.87..69.87 rows=953 width=32) (actual time=0.383..0.384 rows=1002 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 71kB
                                                        Buffers: shared hit=17
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..69.87 rows=953 width=32) (actual time=0.016..0.213 rows=1002 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=17
                          ->  Limit  (cost=138.43..138.43 rows=1 width=1392) (actual time=0.495..0.497 rows=0 loops=1)
                                Buffers: shared hit=84
                                ->  Sort  (cost=138.43..138.43 rows=1 width=1392) (actual time=0.494..0.496 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=84
                                      ->  Result  (cost=82.22..138.42 rows=1 width=1392) (actual time=0.491..0.492 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=84
                                            ->  Hash Semi Join  (cost=82.22..138.42 rows=1 width=1384) (actual time=0.487..0.487 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=84
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.016..0.105 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=69.87..69.87 rows=953 width=32) (actual time=0.370..0.371 rows=1002 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 71kB
                                                        Buffers: shared hit=17
                                                        ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..69.87 rows=953 width=32) (actual time=0.011..0.203 rows=1002 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=17
                          ->  Limit  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.124..0.126 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.123..0.125 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.35 rows=1 width=1392) (actual time=0.098..0.101 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1384) (actual time=0.093..0.095 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.049..0.066 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.037..0.039 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.028..0.032 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.021..0.022 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                        Filter: (value < '25'::double precision)
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=40)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=120
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=40)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=120
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.522 ms
Execution Time: 9.002 ms
```

### num lt 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=7867.45..7867.46 rows=1 width=8) (actual time=11.359..11.362 rows=1 loops=1)
  Buffers: shared hit=4021
  ->  Limit  (cost=75.07..7864.50 rows=236 width=4) (actual time=0.917..11.056 rows=1001 loops=1)
        Buffers: shared hit=4021
        ->  Result  (cost=75.07..7864.50 rows=236 width=4) (actual time=0.916..10.811 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=4021
              ->  Nested Loop  (cost=75.07..7864.50 rows=236 width=0) (actual time=0.904..10.523 rows=1001 loops=1)
                    Buffers: shared hit=4021
                    ->  HashAggregate  (cost=74.63..84.16 rows=953 width=32) (actual time=0.877..1.285 rows=1001 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=17
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..69.87 rows=953 width=32) (actual time=0.037..0.348 rows=1002 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=17
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.16 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=1001)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=4004
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.723 ms
Execution Time: 11.452 ms
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
Limit  (cost=197.08..197.21 rows=51 width=1426) (actual time=13.241..13.261 rows=51 loops=1)
  Buffers: shared hit=7034
  ->  Sort  (cost=197.08..197.67 rows=236 width=1426) (actual time=13.239..13.252 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 193kB
        Buffers: shared hit=7034
        ->  Result  (cost=75.63..189.21 rows=236 width=1426) (actual time=0.774..12.276 rows=1002 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=7034
              ->  Nested Loop Left Join  (cost=75.63..187.44 rows=236 width=1399) (actual time=0.753..11.499 rows=1002 loops=1)
                    Buffers: shared hit=7034
                    ->  Nested Loop  (cost=75.33..110.22 rows=236 width=1391) (actual time=0.729..8.042 rows=1002 loops=1)
                          Buffers: shared hit=4028
                          ->  Nested Loop  (cost=74.90..101.98 rows=1 width=71) (actual time=0.703..1.208 rows=1002 loops=1)
                                Buffers: shared hit=20
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.031..0.036 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=74.63..84.16 rows=953 width=32) (actual time=0.669..0.953 rows=1002 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=17
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..69.87 rows=953 width=32) (actual time=0.027..0.286 rows=1002 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '25'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=17
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.16 rows=1 width=1384) (actual time=0.006..0.006 rows=1 loops=1002)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=4008
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=1002)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=3006
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=1002)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=3006
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.318 ms
Execution Time: 13.402 ms
```

### num lt 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value < $9::double precision) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=796.94..796.95 rows=1 width=8) (actual time=1.072..1.083 rows=1 loops=1)
  Buffers: shared hit=508
  ->  Limit  (cost=5.96..796.89 rows=4 width=4) (actual time=0.362..1.078 rows=3 loops=1)
        Buffers: shared hit=508
        ->  Result  (cost=5.96..796.89 rows=4 width=4) (actual time=0.361..1.076 rows=3 loops=1)
              Buffers: shared hit=508
              ->  Append  (cost=5.96..796.85 rows=4 width=0) (actual time=0.361..1.075 rows=3 loops=1)
                    Buffers: shared hit=508
                    ->  Subquery Scan on "*SELECT* 1"  (cost=5.96..344.45 rows=1 width=0) (actual time=0.359..0.492 rows=3 loops=1)
                          Buffers: shared hit=200
                          ->  Limit  (cost=5.96..344.44 rows=1 width=4) (actual time=0.359..0.490 rows=3 loops=1)
                                Buffers: shared hit=200
                                ->  Result  (cost=5.96..344.44 rows=1 width=4) (actual time=0.358..0.488 rows=3 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=200
                                      ->  Nested Loop  (cost=5.96..344.44 rows=1 width=0) (actual time=0.351..0.480 rows=3 loops=1)
                                            Buffers: shared hit=200
                                            ->  HashAggregate  (cost=5.53..5.93 rows=40 width=32) (actual time=0.059..0.071 rows=49 loops=1)
                                                  Group Key: v.tenant_id, v.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.33 rows=40 width=32) (actual time=0.029..0.036 rows=49 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=49)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=196
                    ->  Subquery Scan on "*SELECT* 2"  (cost=5.96..344.55 rows=1 width=0) (actual time=0.219..0.221 rows=0 loops=1)
                          Buffers: shared hit=200
                          ->  Limit  (cost=5.96..344.54 rows=1 width=4) (actual time=0.218..0.219 rows=0 loops=1)
                                Buffers: shared hit=200
                                ->  Result  (cost=5.96..344.54 rows=1 width=4) (actual time=0.217..0.218 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=200
                                      ->  Nested Loop  (cost=5.96..344.54 rows=1 width=0) (actual time=0.214..0.215 rows=0 loops=1)
                                            Buffers: shared hit=200
                                            ->  HashAggregate  (cost=5.53..5.93 rows=40 width=32) (actual time=0.035..0.045 rows=49 loops=1)
                                                  Group Key: v_1.tenant_id, v_1.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=4
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..5.33 rows=40 width=32) (actual time=0.010..0.016 rows=49 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                                            ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=49)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v_1.record_id))
                                                  Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=196
                    ->  Subquery Scan on "*SELECT* 3"  (cost=6.26..62.47 rows=1 width=0) (actual time=0.242..0.244 rows=0 loops=1)
                          Buffers: shared hit=71
                          ->  Limit  (cost=6.26..62.46 rows=1 width=4) (actual time=0.242..0.243 rows=0 loops=1)
                                Buffers: shared hit=71
                                ->  Result  (cost=6.26..62.46 rows=1 width=4) (actual time=0.241..0.242 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=71
                                      ->  Hash Semi Join  (cost=6.26..62.46 rows=1 width=0) (actual time=0.239..0.240 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=71
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.017..0.197 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=5.33..5.33 rows=40 width=32) (actual time=0.028..0.029 rows=49 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 12kB
                                                  Buffers: shared hit=4
                                                  ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..5.33 rows=40 width=32) (actual time=0.008..0.017 rows=49 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=4
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.36 rows=1 width=0) (actual time=0.111..0.113 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.35 rows=1 width=4) (actual time=0.110..0.113 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.35 rows=1 width=4) (actual time=0.110..0.111 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=0) (actual time=0.106..0.107 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.056..0.072 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.046 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.033..0.039 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.025..0.026 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.010..0.010 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.008..0.008 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                  Filter: (value < '1'::double precision)
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.210 ms
Execution Time: 1.245 ms
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
Nested Loop Left Join  (cost=797.49..813.66 rows=1 width=1434) (actual time=1.120..1.154 rows=3 loops=1)
  Buffers: shared hit=526
  ->  Nested Loop  (cost=797.20..805.33 rows=1 width=1399) (actual time=1.105..1.128 rows=3 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=517
        ->  Subquery Scan on x  (cost=796.91..797.01 rows=1 width=1392) (actual time=1.092..1.108 rows=3 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=508
              ->  Limit  (cost=796.91..796.92 rows=4 width=1392) (actual time=1.089..1.101 rows=3 loops=1)
                    Buffers: shared hit=508
                    ->  Sort  (cost=796.91..796.92 rows=4 width=1392) (actual time=1.087..1.098 rows=3 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Sort Method: quicksort  Memory: 31kB
                          Buffers: shared hit=508
                          ->  Append  (cost=344.45..796.87 rows=4 width=1392) (actual time=0.511..1.091 rows=3 loops=1)
                                Buffers: shared hit=508
                                ->  Limit  (cost=344.45..344.45 rows=1 width=1392) (actual time=0.510..0.513 rows=3 loops=1)
                                      Buffers: shared hit=200
                                      ->  Sort  (cost=344.45..344.45 rows=1 width=1392) (actual time=0.509..0.512 rows=3 loops=1)
                                            Sort Key: r.updated_at DESC, r.id DESC
                                            Sort Method: quicksort  Memory: 31kB
                                            Buffers: shared hit=200
                                            ->  Result  (cost=5.96..344.44 rows=1 width=1392) (actual time=0.375..0.500 rows=3 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=200
                                                  ->  Nested Loop  (cost=5.96..344.44 rows=1 width=1384) (actual time=0.369..0.493 rows=3 loops=1)
                                                        Buffers: shared hit=200
                                                        ->  HashAggregate  (cost=5.53..5.93 rows=40 width=32) (actual time=0.057..0.068 rows=49 loops=1)
                                                              Group Key: v.tenant_id, v.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.33 rows=40 width=32) (actual time=0.028..0.035 rows=49 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.008..0.008 rows=0 loops=49)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                              Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=196
                                ->  Limit  (cost=344.55..344.55 rows=1 width=1392) (actual time=0.227..0.229 rows=0 loops=1)
                                      Buffers: shared hit=200
                                      ->  Sort  (cost=344.55..344.55 rows=1 width=1392) (actual time=0.226..0.228 rows=0 loops=1)
                                            Sort Key: r_1.updated_at DESC, r_1.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=200
                                            ->  Result  (cost=5.96..344.54 rows=1 width=1392) (actual time=0.223..0.224 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=200
                                                  ->  Nested Loop  (cost=5.96..344.54 rows=1 width=1384) (actual time=0.219..0.220 rows=0 loops=1)
                                                        Buffers: shared hit=200
                                                        ->  HashAggregate  (cost=5.53..5.93 rows=40 width=32) (actual time=0.036..0.046 rows=49 loops=1)
                                                              Group Key: v_1.tenant_id, v_1.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_1  (cost=0.42..5.33 rows=40 width=32) (actual time=0.011..0.016 rows=49 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                                        ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.003..0.003 rows=0 loops=49)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v_1.record_id))
                                                              Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=196
                                ->  Limit  (cost=62.47..62.48 rows=1 width=1392) (actual time=0.179..0.181 rows=0 loops=1)
                                      Buffers: shared hit=71
                                      ->  Sort  (cost=62.47..62.48 rows=1 width=1392) (actual time=0.179..0.180 rows=0 loops=1)
                                            Sort Key: r_2.updated_at DESC, r_2.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=71
                                            ->  Result  (cost=6.26..62.46 rows=1 width=1392) (actual time=0.177..0.179 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=71
                                                  ->  Hash Semi Join  (cost=6.26..62.46 rows=1 width=1384) (actual time=0.175..0.176 rows=0 loops=1)
                                                        Hash Cond: (r_2.id = v_2.record_id)
                                                        Buffers: shared hit=71
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.016..0.136 rows=61 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 3
                                                              Buffers: shared hit=67
                                                        ->  Hash  (cost=5.33..5.33 rows=40 width=32) (actual time=0.027..0.027 rows=49 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 12kB
                                                              Buffers: shared hit=4
                                                              ->  Index Only Scan using rfv_num_value on rfv_num v_2  (cost=0.42..5.33 rows=40 width=32) (actual time=0.008..0.016 rows=49 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=4
                                ->  Limit  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.160..0.163 rows=0 loops=1)
                                      Buffers: shared hit=37
                                      ->  Sort  (cost=45.36..45.37 rows=1 width=1392) (actual time=0.159..0.162 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=37
                                            ->  Result  (cost=36.85..45.35 rows=1 width=1392) (actual time=0.157..0.159 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop Semi Join  (cost=36.85..45.35 rows=1 width=1384) (actual time=0.155..0.157 rows=0 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.048..0.067 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.037..0.039 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.029..0.035 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.022 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.009 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_num_pkey on rfv_num v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.029..0.029 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid))
                                                              Filter: (value < '1'::double precision)
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.004..0.004 rows=1 loops=3)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=9
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=3)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=9
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.905 ms
Execution Time: 1.379 ms
```

### num lt 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value < $8::double precision) LIMIT 1001) c
```

```
Aggregate  (cost=344.36..344.37 rows=1 width=8) (actual time=0.453..0.454 rows=1 loops=1)
  Buffers: shared hit=200
  ->  Limit  (cost=5.96..344.24 rows=10 width=4) (actual time=0.080..0.445 rows=49 loops=1)
        Buffers: shared hit=200
        ->  Result  (cost=5.96..344.24 rows=10 width=4) (actual time=0.079..0.437 rows=49 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=200
              ->  Nested Loop  (cost=5.96..344.24 rows=10 width=0) (actual time=0.072..0.423 rows=49 loops=1)
                    Buffers: shared hit=200
                    ->  HashAggregate  (cost=5.53..5.93 rows=40 width=32) (actual time=0.054..0.064 rows=49 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=4
                          ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.33 rows=40 width=32) (actual time=0.028..0.034 rows=49 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                Heap Fetches: 0
                                Buffers: shared hit=4
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=49)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=196
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.473 ms
Execution Time: 0.502 ms
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
Limit  (cost=26.59..26.62 rows=10 width=1426) (actual time=0.849..0.861 rows=49 loops=1)
  Buffers: shared hit=350
  ->  Sort  (cost=26.59..26.62 rows=10 width=1426) (actual time=0.848..0.854 rows=49 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 123kB
        Buffers: shared hit=350
        ->  Result  (cost=6.52..26.43 rows=10 width=1426) (actual time=0.119..0.790 rows=49 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=350
              ->  Nested Loop Left Join  (cost=6.52..26.35 rows=10 width=1399) (actual time=0.105..0.706 rows=49 loops=1)
                    Buffers: shared hit=350
                    ->  Nested Loop  (cost=6.22..23.07 rows=10 width=1391) (actual time=0.091..0.501 rows=49 loops=1)
                          Buffers: shared hit=203
                          ->  Nested Loop  (cost=5.80..14.62 rows=1 width=71) (actual time=0.073..0.098 rows=49 loops=1)
                                Buffers: shared hit=7
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.023..0.025 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=5.53..5.93 rows=40 width=32) (actual time=0.049..0.061 rows=49 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 24kB
                                      Buffers: shared hit=4
                                      ->  Index Only Scan using rfv_num_value on rfv_num v  (cost=0.42..5.33 rows=40 width=32) (actual time=0.022..0.028 rows=49 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = 'b2d0c5f6-d09e-4ee6-b4bb-6afc8305267b'::uuid) AND (value < '1'::double precision))
                                            Heap Fetches: 0
                                            Buffers: shared hit=4
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=49)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=196
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=49)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=147
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=49)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=147
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.872 ms
Execution Time: 0.943 ms
```

### ts range 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=18445.03..18445.04 rows=1 width=8) (actual time=90.671..90.685 rows=1 loops=1)
  Buffers: shared hit=5187, temp written=524
  I/O Timings: temp write=4.144
  ->  Limit  (cost=15124.81..18432.52 rows=1001 width=4) (actual time=85.900..90.558 rows=1001 loops=1)
        Buffers: shared hit=5187, temp written=524
        I/O Timings: temp write=4.144
        ->  Result  (cost=15124.81..18531.65 rows=1031 width=4) (actual time=85.899..90.418 rows=1001 loops=1)
              Buffers: shared hit=5187, temp written=524
              I/O Timings: temp write=4.144
              ->  Append  (cost=15124.81..18521.34 rows=1031 width=0) (actual time=85.898..90.221 rows=1001 loops=1)
                    Buffers: shared hit=5187, temp written=524
                    I/O Timings: temp write=4.144
                    ->  Subquery Scan on "*SELECT* 1"  (cost=15124.81..16600.20 rows=1001 width=0) (actual time=85.896..90.071 rows=1001 loops=1)
                          Buffers: shared hit=5187, temp written=524
                          I/O Timings: temp write=4.144
                          ->  Limit  (cost=15124.81..16590.19 rows=1001 width=4) (actual time=85.895..89.922 rows=1001 loops=1)
                                Buffers: shared hit=5187, temp written=524
                                I/O Timings: temp write=4.144
                                ->  Result  (cost=15124.81..16858.09 rows=1184 width=4) (actual time=85.894..89.810 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=5187, temp written=524
                                      I/O Timings: temp write=4.144
                                      ->  Hash Semi Join  (cost=15124.81..16858.09 rows=1184 width=0) (actual time=85.883..89.638 rows=1001 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=5187, temp written=524
                                            I/O Timings: temp write=4.144
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.037..1.796 rows=5016 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=3260
                                            ->  Hash  (cost=11988.97..11988.97 rows=162102 width=32) (actual time=84.994..84.996 rows=160327 loops=1)
                                                  Buckets: 131072  Batches: 2  Memory Usage: 6032kB
                                                  Buffers: shared hit=1927, temp written=509
                                                  I/O Timings: temp write=4.023
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..11988.97 rows=162102 width=32) (actual time=0.030..38.195 rows=160327 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=1927
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.86..1712.44 rows=27 width=0) (never executed)
                          ->  Limit  (cost=0.86..1712.17 rows=27 width=4) (never executed)
                                ->  Result  (cost=0.86..1712.17 rows=27 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..1712.17 rows=27 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_1.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.18 rows=2 width=0) (never executed)
                          ->  Limit  (cost=0.86..158.16 rows=2 width=4) (never executed)
                                ->  Result  (cost=0.86..158.16 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..8.45 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.37 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.85..45.36 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.85..45.36 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.233 ms
Execution Time: 91.959 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $11
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=48.57..15890.23 rows=1 width=1434) (actual time=0.424..4.176 rows=51 loops=1)
  Buffers: shared hit=2856
  ->  Nested Loop  (cost=48.28..15881.89 rows=1 width=1399) (actual time=0.404..3.950 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2703
        ->  Subquery Scan on x  (cost=47.99..15873.58 rows=1 width=1392) (actual time=0.386..3.795 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2550
              ->  Limit  (cost=47.99..15872.43 rows=51 width=1392) (actual time=0.383..3.739 rows=51 loops=1)
                    Buffers: shared hit=2550
                    ->  Merge Append  (cost=47.99..25180.93 rows=81 width=1392) (actual time=0.381..3.728 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2550
                          ->  Limit  (cost=0.86..23263.98 rows=51 width=1392) (actual time=0.116..3.436 rows=51 loops=1)
                                Buffers: shared hit=2467
                                ->  Result  (cost=0.86..540070.02 rows=1184 width=1392) (actual time=0.115..3.426 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=2467
                                      ->  Nested Loop Semi Join  (cost=0.86..540070.02 rows=1184 width=1384) (actual time=0.111..3.402 rows=51 loops=1)
                                            Buffers: shared hit=2467
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..501702.69 rows=5845 width=1384) (actual time=0.089..2.704 rows=127 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1807
                                                  Buffers: shared hit=1959
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..6.55 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=127)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=508
                          ->  Limit  (cost=0.86..1712.17 rows=27 width=1392) (actual time=0.076..0.080 rows=1 loops=1)
                                Buffers: shared hit=28
                                ->  Result  (cost=0.86..1712.17 rows=27 width=1392) (actual time=0.076..0.078 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=28
                                      ->  Nested Loop Semi Join  (cost=0.86..1712.17 rows=27 width=1384) (actual time=0.073..0.075 rows=1 loops=1)
                                            Buffers: shared hit=28
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.019..0.029 rows=5 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=8
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_1  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=5)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_1.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=20
                          ->  Limit  (cost=0.86..158.16 rows=2 width=1392) (actual time=0.058..0.060 rows=1 loops=1)
                                Buffers: shared hit=18
                                ->  Result  (cost=0.86..158.16 rows=2 width=1392) (actual time=0.057..0.059 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=18
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=2 width=1384) (actual time=0.054..0.056 rows=1 loops=1)
                                            Buffers: shared hit=18
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.019..0.026 rows=3 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=6
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
                          ->  Limit  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.129..0.135 rows=1 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.129..0.134 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.36 rows=1 width=1392) (actual time=0.083..0.123 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=1384) (actual time=0.081..0.120 rows=2 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.068..0.089 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.048..0.052 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.037..0.044 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.031 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.011..0.011 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                        Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.989 ms
Execution Time: 4.538 ms
```

### ts range 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=4515.20..4515.21 rows=1 width=8) (actual time=10.767..10.769 rows=1 loops=1)
  Buffers: shared hit=11271
  ->  Limit  (cost=0.86..4502.69 rows=1001 width=4) (actual time=0.053..10.631 rows=1001 loops=1)
        Buffers: shared hit=11271
        ->  Result  (cost=0.86..180536.99 rows=40143 width=4) (actual time=0.052..10.485 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=11271
              ->  Nested Loop Semi Join  (cost=0.86..180536.99 rows=40143 width=0) (actual time=0.044..10.302 rows=1001 loops=1)
                    Buffers: shared hit=11271
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.025..0.793 rows=2423 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1579
                    ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.83 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=2423)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                          Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                          Rows Removed by Filter: 1
                          Buffers: shared hit=9692
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.489 ms
Execution Time: 10.813 ms
```

### ts range 40%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10
```

```
Limit  (cost=1.42..865.17 rows=51 width=1426) (actual time=0.090..1.547 rows=51 loops=1)
  Buffers: shared hit=721
  ->  Result  (cost=1.42..679870.72 rows=40143 width=1426) (actual time=0.089..1.533 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
        Buffers: shared hit=721
        ->  Nested Loop Left Join  (cost=1.42..679569.64 rows=40143 width=1399) (actual time=0.075..1.473 rows=51 loops=1)
              Buffers: shared hit=721
              ->  Nested Loop  (cost=1.12..666436.88 rows=40143 width=1391) (actual time=0.061..1.155 rows=51 loops=1)
                    Buffers: shared hit=568
                    ->  Nested Loop Semi Join  (cost=0.85..665926.80 rows=40143 width=1384) (actual time=0.047..1.094 rows=51 loops=1)
                          Buffers: shared hit=565
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..500712.14 rows=198111 width=1384) (actual time=0.028..0.294 rows=112 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                Buffers: shared hit=117
                          ->  Index Scan using rfv_ts_pkey on rfv_ts v  (cost=0.42..0.83 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=112)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2025-05-01 02:40:00+00'::timestamp with time zone))
                                Rows Removed by Filter: 1
                                Buffers: shared hit=448
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.010..0.012 rows=1 loops=1)
                                Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.005..0.005 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.835 ms
Execution Time: 1.630 ms
```

### ts range 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2726.22..2726.23 rows=1 width=8) (actual time=19.752..19.765 rows=1 loops=1)
  Buffers: shared hit=16754
  ->  Limit  (cost=713.69..2725.43 rows=63 width=4) (actual time=3.422..19.693 rows=464 loops=1)
        Buffers: shared hit=16754
        ->  Result  (cost=713.69..2725.43 rows=63 width=4) (actual time=3.421..19.621 rows=464 loops=1)
              Buffers: shared hit=16754
              ->  Append  (cost=713.69..2724.80 rows=63 width=0) (actual time=3.420..19.530 rows=464 loops=1)
                    Buffers: shared hit=16754
                    ->  Subquery Scan on "*SELECT* 1"  (cost=713.69..1246.06 rows=60 width=0) (actual time=3.419..14.484 rows=459 loops=1)
                          Buffers: shared hit=15783
                          ->  Limit  (cost=713.69..1245.46 rows=60 width=4) (actual time=3.417..14.404 rows=459 loops=1)
                                Buffers: shared hit=15783
                                ->  Result  (cost=713.69..1245.46 rows=60 width=4) (actual time=3.417..14.340 rows=459 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15783
                                      ->  Hash Semi Join  (cost=713.69..1245.46 rows=60 width=0) (actual time=3.408..14.254 rows=459 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15783
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.030..7.264 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=610.23..610.23 rows=8232 width=32) (actual time=3.347..3.348 rows=8159 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 638kB
                                                  Buffers: shared hit=102
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..610.23 rows=8232 width=32) (actual time=0.017..1.834 rows=8159 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=102
                    ->  Subquery Scan on "*SELECT* 2"  (cost=727.68..1274.89 rows=1 width=0) (actual time=3.455..4.300 rows=4 loops=1)
                          Buffers: shared hit=623
                          ->  Limit  (cost=727.68..1274.88 rows=1 width=4) (actual time=3.454..4.298 rows=4 loops=1)
                                Buffers: shared hit=623
                                ->  Result  (cost=727.68..1274.88 rows=1 width=4) (actual time=3.451..4.294 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=623
                                      ->  Hash Semi Join  (cost=727.68..1274.88 rows=1 width=0) (actual time=3.441..4.282 rows=4 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=623
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.228..1.010 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.136..0.136 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=610.23..610.23 rows=8232 width=32) (actual time=3.169..3.169 rows=8159 loops=1)
                                                  Buckets: 16384  Batches: 1  Memory Usage: 638kB
                                                  Buffers: shared hit=102
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..610.23 rows=8232 width=32) (actual time=0.016..1.688 rows=8159 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=102
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.86..158.17 rows=1 width=0) (actual time=0.132..0.550 rows=1 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.86..158.16 rows=1 width=4) (actual time=0.131..0.548 rows=1 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=4) (actual time=0.130..0.546 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=0) (actual time=0.099..0.515 rows=1 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.026..0.120 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.37 rows=1 width=0) (actual time=0.097..0.100 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.36 rows=1 width=4) (actual time=0.096..0.098 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.36 rows=1 width=4) (actual time=0.095..0.097 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=0) (actual time=0.090..0.093 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.056..0.073 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.044..0.047 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.036 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.041 ms
Execution Time: 19.923 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $11
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=22330.44..22497.11 rows=1 width=1434) (actual time=56.608..57.399 rows=51 loops=1)
  Buffers: shared hit=23176
  ->  Nested Loop  (cost=22330.15..22488.78 rows=1 width=1399) (actual time=56.587..57.228 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=23023
        ->  Subquery Scan on x  (cost=22329.86..22480.46 rows=1 width=1392) (actual time=56.571..57.099 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22870
              ->  Limit  (cost=22329.86..22479.32 rows=51 width=1392) (actual time=56.568..57.059 rows=51 loops=1)
                    Buffers: shared hit=22870
                    ->  Merge Append  (cost=22329.86..22488.11 rows=54 width=1392) (actual time=56.567..57.052 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22870
                          ->  Limit  (cost=21008.70..21008.83 rows=51 width=1392) (actual time=51.797..51.821 rows=49 loops=1)
                                Buffers: shared hit=21899
                                ->  Sort  (cost=21008.70..21008.85 rows=60 width=1392) (actual time=51.795..51.814 rows=49 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 197kB
                                      Buffers: shared hit=21899
                                      ->  Result  (cost=1172.46..21006.93 rows=60 width=1392) (actual time=12.632..51.309 rows=459 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=21899
                                            ->  Hash Semi Join  (cost=1172.46..21006.93 rows=60 width=1384) (actual time=12.626..51.151 rows=459 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21899
                                                  ->  Bitmap Heap Scan on records r  (cost=459.32..20277.78 rows=5845 width=1384) (actual time=9.404..43.040 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.86 rows=5845 width=0) (actual time=4.554..4.554 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=610.23..610.23 rows=8232 width=32) (actual time=3.115..3.116 rows=8159 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 638kB
                                                        Buffers: shared hit=102
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..610.23 rows=8232 width=32) (actual time=0.022..1.684 rows=8159 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=102
                          ->  Limit  (cost=1274.89..1274.90 rows=1 width=1392) (actual time=4.512..4.516 rows=2 loops=1)
                                Buffers: shared hit=623
                                ->  Sort  (cost=1274.89..1274.90 rows=1 width=1392) (actual time=4.509..4.513 rows=2 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 33kB
                                      Buffers: shared hit=623
                                      ->  Result  (cost=727.68..1274.88 rows=1 width=1392) (actual time=3.522..4.488 rows=4 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=623
                                            ->  Hash Semi Join  (cost=727.68..1274.88 rows=1 width=1384) (actual time=3.508..4.471 rows=4 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=623
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.220..1.116 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.106..0.106 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=610.23..610.23 rows=8232 width=32) (actual time=3.242..3.243 rows=8159 loops=1)
                                                        Buckets: 16384  Batches: 1  Memory Usage: 638kB
                                                        Buffers: shared hit=102
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..610.23 rows=8232 width=32) (actual time=0.020..1.637 rows=8159 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=102
                          ->  Limit  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.135..0.564 rows=1 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.86..158.16 rows=1 width=1392) (actual time=0.133..0.562 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.86..158.16 rows=1 width=1384) (actual time=0.123..0.550 rows=1 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.045..0.152 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_2  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_2.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.116..0.121 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.116..0.120 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.36 rows=1 width=1392) (actual time=0.107..0.111 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=1384) (actual time=0.104..0.107 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.062..0.078 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.045..0.048 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.034..0.039 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.014 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.008..0.008 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                        Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.602 ms
Execution Time: 57.656 ms
```

### ts range 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=7738.94..7738.95 rows=1 width=8) (actual time=23.482..23.486 rows=1 loops=1)
  Buffers: shared hit=31864
  ->  Limit  (cost=713.56..7726.43 rows=1001 width=4) (actual time=3.312..23.370 rows=1001 loops=1)
        Buffers: shared hit=31864
        ->  Result  (cost=713.56..14998.51 rows=2039 width=4) (actual time=3.310..23.242 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=31864
              ->  Hash Semi Join  (cost=713.56..14998.51 rows=2039 width=0) (actual time=3.302..23.077 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=31864
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.026..13.069 rows=49004 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=31762
                    ->  Hash  (cost=610.23..610.23 rows=8232 width=32) (actual time=3.258..3.259 rows=8159 loops=1)
                          Buckets: 16384  Batches: 1  Memory Usage: 638kB
                          Buffers: shared hit=102
                          ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..610.23 rows=8232 width=32) (actual time=0.017..1.754 rows=8159 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                Heap Fetches: 0
                                Buffers: shared hit=102
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.516 ms
Execution Time: 23.537 ms
```

### ts range 2%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10
```

```
Limit  (cost=1064.98..1065.10 rows=51 width=1426) (actual time=71.882..71.904 rows=51 loops=1)
  Buffers: shared hit=32795
  ->  Sort  (cost=1064.98..1070.07 rows=2039 width=1426) (actual time=71.880..71.897 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 199kB
        Buffers: shared hit=32795
        ->  Result  (cost=796.79..996.95 rows=2039 width=1426) (actual time=4.347..65.525 rows=8159 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32795
              ->  Hash Left Join  (cost=796.79..981.66 rows=2039 width=1399) (actual time=4.334..60.320 rows=8159 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32795
                    ->  Nested Loop  (cost=652.08..831.60 rows=2039 width=1391) (actual time=4.173..56.701 rows=8159 loops=1)
                          Buffers: shared hit=32741
                          ->  Nested Loop  (cost=651.66..824.32 rows=1 width=71) (actual time=4.141..8.176 rows=8159 loops=1)
                                Buffers: shared hit=105
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.017..0.022 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=651.38..733.71 rows=8232 width=32) (actual time=4.121..6.664 rows=8159 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=102
                                      ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..610.23 rows=8232 width=32) (actual time=0.016..1.524 rows=8159 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-08-10 05:20:00+00'::timestamp with time zone))
                                            Heap Fetches: 0
                                            Buffers: shared hit=102
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.78 rows=1 width=1384) (actual time=0.005..0.005 rows=1 loops=8159)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=32636
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.148..0.150 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.037..0.137 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.025..0.116 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.012..0.013 rows=53 loops=1)
                                            Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.816 ms
Execution Time: 72.030 ms
```

### ts range 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1466.76..1466.77 rows=1 width=8) (actual time=15.380..15.395 rows=1 loops=1)
  Buffers: shared hit=16360
  ->  Limit  (cost=90.86..1466.64 rows=10 width=4) (actual time=0.871..15.373 rows=69 loops=1)
        Buffers: shared hit=16360
        ->  Result  (cost=90.86..1466.64 rows=10 width=4) (actual time=0.870..15.356 rows=69 loops=1)
              Buffers: shared hit=16360
              ->  Append  (cost=90.86..1466.54 rows=10 width=0) (actual time=0.869..15.335 rows=69 loops=1)
                    Buffers: shared hit=16360
                    ->  Subquery Scan on "*SELECT* 1"  (cost=90.86..622.11 rows=7 width=0) (actual time=0.868..12.790 rows=68 loops=1)
                          Buffers: shared hit=15699
                          ->  Limit  (cost=90.86..622.04 rows=7 width=4) (actual time=0.867..12.770 rows=68 loops=1)
                                Buffers: shared hit=15699
                                ->  Result  (cost=90.86..622.04 rows=7 width=4) (actual time=0.866..12.754 rows=68 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15699
                                      ->  Hash Semi Join  (cost=90.86..622.04 rows=7 width=0) (actual time=0.856..12.724 rows=68 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15699
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.033..8.609 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=77.67..77.67 rows=1010 width=32) (actual time=0.489..0.490 rows=1042 loops=1)
                                                  Buckets: 2048 (originally 1024)  Batches: 1 (originally 1)  Memory Usage: 82kB
                                                  Buffers: shared hit=18
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.019..0.269 rows=1042 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=18
                    ->  Subquery Scan on "*SELECT* 2"  (cost=104.86..652.06 rows=1 width=0) (actual time=0.785..1.758 rows=1 loops=1)
                          Buffers: shared hit=539
                          ->  Limit  (cost=104.86..652.05 rows=1 width=4) (actual time=0.784..1.756 rows=1 loops=1)
                                Buffers: shared hit=539
                                ->  Result  (cost=104.86..652.05 rows=1 width=4) (actual time=0.781..1.752 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=539
                                      ->  Hash Semi Join  (cost=104.86..652.05 rows=1 width=0) (actual time=0.765..1.735 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=539
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.230..1.154 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.126..0.127 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=77.67..77.67 rows=1010 width=32) (actual time=0.500..0.501 rows=1042 loops=1)
                                                  Buckets: 2048 (originally 1024)  Batches: 1 (originally 1)  Memory Usage: 82kB
                                                  Buffers: shared hit=18
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.018..0.259 rows=1042 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=18
                    ->  Subquery Scan on "*SELECT* 3"  (cost=90.74..146.95 rows=1 width=0) (actual time=0.631..0.632 rows=0 loops=1)
                          Buffers: shared hit=85
                          ->  Limit  (cost=90.74..146.94 rows=1 width=4) (actual time=0.629..0.631 rows=0 loops=1)
                                Buffers: shared hit=85
                                ->  Result  (cost=90.74..146.94 rows=1 width=4) (actual time=0.628..0.629 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=85
                                      ->  Hash Semi Join  (cost=90.74..146.94 rows=1 width=0) (actual time=0.618..0.619 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=85
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.023..0.132 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=77.67..77.67 rows=1010 width=32) (actual time=0.471..0.472 rows=1042 loops=1)
                                                  Buckets: 2048 (originally 1024)  Batches: 1 (originally 1)  Memory Usage: 82kB
                                                  Buffers: shared hit=18
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v_2  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.014..0.253 rows=1042 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=18
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.37 rows=1 width=0) (actual time=0.125..0.128 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.36 rows=1 width=4) (actual time=0.125..0.127 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.36 rows=1 width=4) (actual time=0.124..0.126 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=0) (actual time=0.121..0.123 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.069..0.089 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.051..0.053 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.036..0.041 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.027..0.028 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.011..0.011 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.015..0.015 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.263 ms
Execution Time: 15.587 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $11
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=9171.60..9188.08 rows=1 width=1434) (actual time=10.213..10.580 rows=51 loops=1)
  Buffers: shared hit=5153
  ->  Nested Loop  (cost=9171.30..9179.74 rows=1 width=1399) (actual time=10.191..10.407 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=5000
        ->  Subquery Scan on x  (cost=9171.02..9171.43 rows=1 width=1392) (actual time=10.174..10.269 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4847
              ->  Limit  (cost=9171.02..9171.20 rows=10 width=1392) (actual time=10.171..10.227 rows=51 loops=1)
                    Buffers: shared hit=4847
                    ->  Merge Append  (cost=9171.02..9171.20 rows=10 width=1392) (actual time=10.169..10.218 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4847
                          ->  Limit  (cost=8326.61..8326.62 rows=7 width=1392) (actual time=7.616..7.637 rows=51 loops=1)
                                Buffers: shared hit=4186
                                ->  Sort  (cost=8326.61..8326.62 rows=7 width=1392) (actual time=7.615..7.629 rows=51 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 161kB
                                      Buffers: shared hit=4186
                                      ->  Result  (cost=83.16..8326.51 rows=7 width=1392) (actual time=0.865..7.431 rows=68 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=4186
                                            ->  Nested Loop  (cost=83.16..8326.51 rows=7 width=1384) (actual time=0.860..7.398 rows=68 loops=1)
                                                  Buffers: shared hit=4186
                                                  ->  HashAggregate  (cost=82.72..92.82 rows=1010 width=32) (actual time=0.599..0.854 rows=1042 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 193kB
                                                        Buffers: shared hit=18
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.031..0.238 rows=1042 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=18
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.15 rows=1 width=1384) (actual time=0.006..0.006 rows=0 loops=1042)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=4168
                          ->  Limit  (cost=652.06..652.06 rows=1 width=1392) (actual time=1.742..1.746 rows=1 loops=1)
                                Buffers: shared hit=539
                                ->  Sort  (cost=652.06..652.06 rows=1 width=1392) (actual time=1.740..1.743 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=539
                                      ->  Result  (cost=104.86..652.05 rows=1 width=1392) (actual time=0.800..1.731 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=539
                                            ->  Hash Semi Join  (cost=104.86..652.05 rows=1 width=1384) (actual time=0.791..1.720 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=539
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.272..1.155 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.171..0.172 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=77.67..77.67 rows=1010 width=32) (actual time=0.489..0.490 rows=1042 loops=1)
                                                        Buckets: 2048 (originally 1024)  Batches: 1 (originally 1)  Memory Usage: 82kB
                                                        Buffers: shared hit=18
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.016..0.244 rows=1042 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=18
                          ->  Limit  (cost=146.95..146.95 rows=1 width=1392) (actual time=0.686..0.687 rows=0 loops=1)
                                Buffers: shared hit=85
                                ->  Sort  (cost=146.95..146.95 rows=1 width=1392) (actual time=0.684..0.686 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=85
                                      ->  Result  (cost=90.74..146.94 rows=1 width=1392) (actual time=0.661..0.662 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=85
                                            ->  Hash Semi Join  (cost=90.74..146.94 rows=1 width=1384) (actual time=0.654..0.655 rows=0 loops=1)
                                                  Hash Cond: (r_2.id = v_2.record_id)
                                                  Buffers: shared hit=85
                                                  ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.025..0.126 rows=61 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Rows Removed by Filter: 3
                                                        Buffers: shared hit=67
                                                  ->  Hash  (cost=77.67..77.67 rows=1010 width=32) (actual time=0.509..0.510 rows=1042 loops=1)
                                                        Buckets: 2048 (originally 1024)  Batches: 1 (originally 1)  Memory Usage: 82kB
                                                        Buffers: shared hit=18
                                                        ->  Index Only Scan using rfv_ts_value on rfv_ts v_2  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.017..0.247 rows=1042 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=18
                          ->  Limit  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.119..0.122 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.118..0.121 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.85..45.36 rows=1 width=1392) (actual time=0.114..0.117 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=1384) (actual time=0.110..0.113 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.064..0.081 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.046..0.049 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.036..0.040 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.029..0.030 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.015 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.013..0.013 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.010..0.010 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                        Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.377 ms
Execution Time: 10.847 ms
```

### ts range 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=8322.39..8322.40 rows=1 width=8) (actual time=6.735..6.737 rows=1 loops=1)
  Buffers: shared hit=4022
  ->  Limit  (cost=83.16..8319.26 rows=250 width=4) (actual time=0.537..6.599 rows=1001 loops=1)
        Buffers: shared hit=4022
        ->  Result  (cost=83.16..8319.26 rows=250 width=4) (actual time=0.536..6.449 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=4022
              ->  Nested Loop  (cost=83.16..8319.26 rows=250 width=0) (actual time=0.530..6.278 rows=1001 loops=1)
                    Buffers: shared hit=4022
                    ->  HashAggregate  (cost=82.72..92.82 rows=1010 width=32) (actual time=0.514..0.727 rows=1001 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 193kB
                          Buffers: shared hit=18
                          ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.023..0.206 rows=1042 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                Heap Fetches: 0
                                Buffers: shared hit=18
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.14 rows=1 width=32) (actual time=0.005..0.005 rows=1 loops=1001)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=4004
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.450 ms
Execution Time: 6.789 ms
```

### ts range 0.25%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10
```

```
Limit  (cost=211.45..211.58 rows=51 width=1426) (actual time=12.074..12.088 rows=51 loops=1)
  Buffers: shared hit=7315
  ->  Sort  (cost=211.45..212.08 rows=250 width=1426) (actual time=12.073..12.081 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 199kB
        Buffers: shared hit=7315
        ->  Result  (cost=83.72..203.11 rows=250 width=1426) (actual time=0.597..11.203 rows=1042 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=7315
              ->  Nested Loop Left Join  (cost=83.72..201.24 rows=250 width=1399) (actual time=0.584..10.510 rows=1042 loops=1)
                    Buffers: shared hit=7315
                    ->  Nested Loop  (cost=83.42..119.44 rows=250 width=1391) (actual time=0.571..7.247 rows=1042 loops=1)
                          Buffers: shared hit=4189
                          ->  Nested Loop  (cost=83.00..111.22 rows=1 width=71) (actual time=0.555..1.026 rows=1042 loops=1)
                                Buffers: shared hit=21
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.020..0.022 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=82.72..92.82 rows=1010 width=32) (actual time=0.534..0.795 rows=1042 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 193kB
                                      Buffers: shared hit=18
                                      ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..77.67 rows=1010 width=32) (actual time=0.016..0.200 rows=1042 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-29 01:40:00+00'::timestamp with time zone))
                                            Heap Fetches: 0
                                            Buffers: shared hit=18
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.14 rows=1 width=1384) (actual time=0.005..0.005 rows=1 loops=1042)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=4168
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=1042)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=3126
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=1042)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=3126
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.964 ms
Execution Time: 12.177 ms
```

### ts range 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=797.25..797.26 rows=1 width=8) (actual time=0.870..0.879 rows=1 loops=1)
  Buffers: shared hit=487
  ->  Limit  (cost=6.06..797.20 rows=4 width=4) (actual time=0.177..0.875 rows=4 loops=1)
        Buffers: shared hit=487
        ->  Result  (cost=6.06..797.20 rows=4 width=4) (actual time=0.175..0.872 rows=4 loops=1)
              Buffers: shared hit=487
              ->  Append  (cost=6.06..797.16 rows=4 width=0) (actual time=0.173..0.869 rows=4 loops=1)
                    Buffers: shared hit=487
                    ->  Subquery Scan on "*SELECT* 1"  (cost=6.06..344.55 rows=1 width=0) (actual time=0.173..0.442 rows=4 loops=1)
                          Buffers: shared hit=189
                          ->  Limit  (cost=6.06..344.54 rows=1 width=4) (actual time=0.172..0.441 rows=4 loops=1)
                                Buffers: shared hit=189
                                ->  Result  (cost=6.06..344.54 rows=1 width=4) (actual time=0.172..0.439 rows=4 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=189
                                      ->  Nested Loop  (cost=6.06..344.54 rows=1 width=0) (actual time=0.164..0.431 rows=4 loops=1)
                                            Buffers: shared hit=189
                                            ->  HashAggregate  (cost=5.62..6.03 rows=40 width=32) (actual time=0.051..0.062 rows=46 loops=1)
                                                  Group Key: v.tenant_id, v.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..5.42 rows=40 width=32) (actual time=0.024..0.033 rows=46 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=46)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=184
                    ->  Subquery Scan on "*SELECT* 2"  (cost=6.06..344.65 rows=1 width=0) (actual time=0.171..0.172 rows=0 loops=1)
                          Buffers: shared hit=189
                          ->  Limit  (cost=6.06..344.64 rows=1 width=4) (actual time=0.170..0.171 rows=0 loops=1)
                                Buffers: shared hit=189
                                ->  Result  (cost=6.06..344.64 rows=1 width=4) (actual time=0.169..0.170 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=189
                                      ->  Nested Loop  (cost=6.06..344.64 rows=1 width=0) (actual time=0.166..0.167 rows=0 loops=1)
                                            Buffers: shared hit=189
                                            ->  HashAggregate  (cost=5.62..6.03 rows=40 width=32) (actual time=0.031..0.039 rows=46 loops=1)
                                                  Group Key: v_1.tenant_id, v_1.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..5.42 rows=40 width=32) (actual time=0.009..0.015 rows=46 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=46)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v_1.record_id))
                                                  Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=184
                    ->  Subquery Scan on "*SELECT* 3"  (cost=6.36..62.57 rows=1 width=0) (actual time=0.161..0.162 rows=0 loops=1)
                          Buffers: shared hit=72
                          ->  Limit  (cost=6.36..62.56 rows=1 width=4) (actual time=0.160..0.161 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Result  (cost=6.36..62.56 rows=1 width=4) (actual time=0.160..0.161 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=72
                                      ->  Hash Semi Join  (cost=6.36..62.56 rows=1 width=0) (actual time=0.158..0.159 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=72
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.015..0.123 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=5.42..5.42 rows=40 width=32) (actual time=0.024..0.024 rows=46 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_ts_value on rfv_ts v_2  (cost=0.42..5.42 rows=40 width=32) (actual time=0.006..0.014 rows=46 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.85..45.37 rows=1 width=0) (actual time=0.088..0.090 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.85..45.36 rows=1 width=4) (actual time=0.087..0.089 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.85..45.36 rows=1 width=4) (actual time=0.086..0.088 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=0) (actual time=0.085..0.086 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.044..0.059 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.035..0.037 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.028..0.033 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.012 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                  Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.885 ms
Execution Time: 1.015 ms
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
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
UNION ALL
(SELECT r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key FROM records r
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value >= $9::timestamptz AND v.value < $10::timestamptz) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $11)
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $11
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC
```

```
Nested Loop Left Join  (cost=797.80..813.96 rows=1 width=1434) (actual time=0.881..0.918 rows=4 loops=1)
  Buffers: shared hit=511
  ->  Nested Loop  (cost=797.50..805.63 rows=1 width=1399) (actual time=0.866..0.890 rows=4 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=499
        ->  Subquery Scan on x  (cost=797.22..797.32 rows=1 width=1392) (actual time=0.855..0.870 rows=4 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=487
              ->  Limit  (cost=797.22..797.23 rows=4 width=1392) (actual time=0.853..0.864 rows=4 loops=1)
                    Buffers: shared hit=487
                    ->  Sort  (cost=797.22..797.23 rows=4 width=1392) (actual time=0.852..0.862 rows=4 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Sort Method: quicksort  Memory: 33kB
                          Buffers: shared hit=487
                          ->  Append  (cost=344.55..797.18 rows=4 width=1392) (actual time=0.422..0.854 rows=4 loops=1)
                                Buffers: shared hit=487
                                ->  Limit  (cost=344.55..344.55 rows=1 width=1392) (actual time=0.421..0.424 rows=4 loops=1)
                                      Buffers: shared hit=189
                                      ->  Sort  (cost=344.55..344.55 rows=1 width=1392) (actual time=0.420..0.422 rows=4 loops=1)
                                            Sort Key: r.updated_at DESC, r.id DESC
                                            Sort Method: quicksort  Memory: 33kB
                                            Buffers: shared hit=189
                                            ->  Result  (cost=6.06..344.54 rows=1 width=1392) (actual time=0.165..0.412 rows=4 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=189
                                                  ->  Nested Loop  (cost=6.06..344.54 rows=1 width=1384) (actual time=0.161..0.407 rows=4 loops=1)
                                                        Buffers: shared hit=189
                                                        ->  HashAggregate  (cost=5.62..6.03 rows=40 width=32) (actual time=0.080..0.090 rows=46 loops=1)
                                                              Group Key: v.tenant_id, v.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=5
                                                              ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..5.42 rows=40 width=32) (actual time=0.022..0.033 rows=46 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5
                                                        ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.007..0.007 rows=0 loops=46)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                              Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=184
                                ->  Limit  (cost=344.65..344.65 rows=1 width=1392) (actual time=0.169..0.171 rows=0 loops=1)
                                      Buffers: shared hit=189
                                      ->  Sort  (cost=344.65..344.65 rows=1 width=1392) (actual time=0.169..0.170 rows=0 loops=1)
                                            Sort Key: r_1.updated_at DESC, r_1.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=189
                                            ->  Result  (cost=6.06..344.64 rows=1 width=1392) (actual time=0.166..0.167 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=189
                                                  ->  Nested Loop  (cost=6.06..344.64 rows=1 width=1384) (actual time=0.163..0.164 rows=0 loops=1)
                                                        Buffers: shared hit=189
                                                        ->  HashAggregate  (cost=5.62..6.03 rows=40 width=32) (actual time=0.028..0.036 rows=46 loops=1)
                                                              Group Key: v_1.tenant_id, v_1.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=5
                                                              ->  Index Only Scan using rfv_ts_value on rfv_ts v_1  (cost=0.42..5.42 rows=40 width=32) (actual time=0.007..0.014 rows=46 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5
                                                        ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.003..0.003 rows=0 loops=46)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v_1.record_id))
                                                              Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=184
                                ->  Limit  (cost=62.57..62.58 rows=1 width=1392) (actual time=0.164..0.165 rows=0 loops=1)
                                      Buffers: shared hit=72
                                      ->  Sort  (cost=62.57..62.58 rows=1 width=1392) (actual time=0.163..0.164 rows=0 loops=1)
                                            Sort Key: r_2.updated_at DESC, r_2.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=72
                                            ->  Result  (cost=6.36..62.56 rows=1 width=1392) (actual time=0.161..0.163 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=72
                                                  ->  Hash Semi Join  (cost=6.36..62.56 rows=1 width=1384) (actual time=0.160..0.161 rows=0 loops=1)
                                                        Hash Cond: (r_2.id = v_2.record_id)
                                                        Buffers: shared hit=72
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.015..0.126 rows=61 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 3
                                                              Buffers: shared hit=67
                                                        ->  Hash  (cost=5.42..5.42 rows=40 width=32) (actual time=0.023..0.024 rows=46 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                              Buffers: shared hit=5
                                                              ->  Index Only Scan using rfv_ts_value on rfv_ts v_2  (cost=0.42..5.42 rows=40 width=32) (actual time=0.006..0.014 rows=46 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=5
                                ->  Limit  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.087..0.089 rows=0 loops=1)
                                      Buffers: shared hit=37
                                      ->  Sort  (cost=45.37..45.37 rows=1 width=1392) (actual time=0.086..0.089 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=37
                                            ->  Result  (cost=36.85..45.36 rows=1 width=1392) (actual time=0.084..0.087 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop Semi Join  (cost=36.85..45.36 rows=1 width=1384) (actual time=0.083..0.085 rows=0 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.044..0.059 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.033..0.035 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.027..0.032 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.020..0.021 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.008..0.008 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.011..0.011 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.007..0.007 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Scan using rfv_ts_pkey on rfv_ts v_3  (cost=0.42..0.88 rows=1 width=32) (actual time=0.008..0.008 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (record_id = r_3.id) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid))
                                                              Filter: ((value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=4)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=12
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=4)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 3.388 ms
Execution Time: 1.204 ms
```

### ts range 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) LIMIT 1001) c
```

```
Aggregate  (cost=344.46..344.47 rows=1 width=8) (actual time=0.348..0.350 rows=1 loops=1)
  Buffers: shared hit=189
  ->  Limit  (cost=6.06..344.34 rows=10 width=4) (actual time=0.059..0.344 rows=46 loops=1)
        Buffers: shared hit=189
        ->  Result  (cost=6.06..344.34 rows=10 width=4) (actual time=0.059..0.339 rows=46 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=189
              ->  Nested Loop  (cost=6.06..344.34 rows=10 width=0) (actual time=0.055..0.330 rows=46 loops=1)
                    Buffers: shared hit=189
                    ->  HashAggregate  (cost=5.62..6.03 rows=40 width=32) (actual time=0.039..0.046 rows=46 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..5.42 rows=40 width=32) (actual time=0.019..0.026 rows=46 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.006..0.006 rows=1 loops=46)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=184
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.328 ms
Execution Time: 0.385 ms
```

### ts range 0.01%, no ACL

```sql
SELECT r.tenant_id, r.id, r.entity_definition_id, d.key, r.version, r.schema_version, r.number, r.state,
       r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
       r.data, r.causation_origin_id, coalesce(r.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       r.updated_at::text FROM records r
JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_ts v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value >= $8::timestamptz AND v.value < $9::timestamptz) ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $10
```

```
Limit  (cost=26.69..26.72 rows=10 width=1426) (actual time=0.554..0.563 rows=46 loops=1)
  Buffers: shared hit=330
  ->  Sort  (cost=26.69..26.72 rows=10 width=1426) (actual time=0.553..0.558 rows=46 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 117kB
        Buffers: shared hit=330
        ->  Result  (cost=6.62..26.53 rows=10 width=1426) (actual time=0.080..0.515 rows=46 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=330
              ->  Nested Loop Left Join  (cost=6.62..26.45 rows=10 width=1399) (actual time=0.071..0.487 rows=46 loops=1)
                    Buffers: shared hit=330
                    ->  Nested Loop  (cost=6.32..23.17 rows=10 width=1391) (actual time=0.060..0.352 rows=46 loops=1)
                          Buffers: shared hit=192
                          ->  Nested Loop  (cost=5.90..14.72 rows=1 width=71) (actual time=0.047..0.061 rows=46 loops=1)
                                Buffers: shared hit=8
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.014..0.015 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=5.62..6.03 rows=40 width=32) (actual time=0.032..0.039 rows=46 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 24kB
                                      Buffers: shared hit=5
                                      ->  Index Only Scan using rfv_ts_value on rfv_ts v  (cost=0.42..5.42 rows=40 width=32) (actual time=0.014..0.020 rows=46 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '79683dd8-2959-491c-bc4e-3760b12bd94d'::uuid) AND (value >= '2024-07-27 08:00:00+00'::timestamp with time zone) AND (value < '2024-07-27 09:40:00+00'::timestamp with time zone))
                                            Heap Fetches: 0
                                            Buffers: shared hit=5
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.006..0.006 rows=1 loops=46)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=184
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=46)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=138
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=46)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=138
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.559 ms
Execution Time: 0.625 ms
```

### bool eq 40%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=12104.22..12104.23 rows=1 width=8) (actual time=27.252..27.263 rows=1 loops=1)
  Buffers: shared hit=15902
  ->  Limit  (cost=942.21..12091.70 rows=1001 width=4) (actual time=19.583..27.125 rows=1001 loops=1)
        Buffers: shared hit=15902
        ->  Result  (cost=942.21..12425.86 rows=1031 width=4) (actual time=19.582..26.971 rows=1001 loops=1)
              Buffers: shared hit=15902
              ->  Append  (cost=942.21..12415.55 rows=1031 width=0) (actual time=19.580..26.799 rows=1001 loops=1)
                    Buffers: shared hit=15902
                    ->  Subquery Scan on "*SELECT* 1"  (cost=942.21..11063.07 rows=1001 width=0) (actual time=19.579..26.661 rows=1001 loops=1)
                          Buffers: shared hit=15902
                          ->  Limit  (cost=942.21..11053.06 rows=1001 width=4) (actual time=19.577..26.505 rows=1001 loops=1)
                                Buffers: shared hit=15902
                                ->  Result  (cost=942.21..12790.38 rows=1173 width=4) (actual time=19.576..26.380 rows=1001 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15902
                                      ->  Merge Semi Join  (cost=942.21..12790.38 rows=1173 width=0) (actual time=19.564..26.180 rows=1001 loops=1)
                                            Merge Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15902
                                            ->  Sort  (cost=882.00..896.61 rows=5845 width=32) (actual time=19.467..20.116 rows=2474 loops=1)
                                                  Sort Key: r.id
                                                  Sort Method: quicksort  Memory: 2087kB
                                                  Buffers: shared hit=15681
                                                  ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.050..9.042 rows=24118 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=15681
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..11525.60 rows=160580 width=32) (actual time=0.060..3.601 rows=17463 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=221
                    ->  Subquery Scan on "*SELECT* 2"  (cost=0.99..1189.65 rows=27 width=0) (never executed)
                          ->  Limit  (cost=0.99..1189.38 rows=27 width=4) (never executed)
                                ->  Result  (cost=0.99..1189.38 rows=27 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.38 rows=27 width=0) (never executed)
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.71 rows=2 width=0) (never executed)
                          ->  Limit  (cost=0.99..111.69 rows=2 width=4) (never executed)
                                ->  Result  (cost=0.99..111.69 rows=2 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=0.99..111.69 rows=2 width=0) (never executed)
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.97 rows=1 width=0) (never executed)
                          ->  Limit  (cost=36.98..45.96 rows=1 width=4) (never executed)
                                ->  Result  (cost=36.98..45.96 rows=1 width=4) (never executed)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=0) (never executed)
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (never executed)
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (never executed)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (never executed)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (never executed)
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (never executed)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (never executed)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (never executed)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.289 ms
Execution Time: 27.494 ms
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
Nested Loop Left Join  (cost=49.57..15309.98 rows=1 width=1434) (actual time=0.524..4.877 rows=51 loops=1)
  Buffers: shared hit=2666
  ->  Nested Loop  (cost=49.27..15301.65 rows=1 width=1399) (actual time=0.499..4.645 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=2513
        ->  Subquery Scan on x  (cost=48.99..15293.33 rows=1 width=1392) (actual time=0.483..4.470 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=2360
              ->  Limit  (cost=48.99..15292.19 rows=51 width=1392) (actual time=0.479..4.403 rows=51 loops=1)
                    Buffers: shared hit=2360
                    ->  Merge Append  (cost=48.99..24258.78 rows=81 width=1392) (actual time=0.478..4.390 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=2360
                          ->  Limit  (cost=0.99..22910.48 rows=51 width=1392) (actual time=0.155..3.994 rows=50 loops=1)
                                Buffers: shared hit=2279
                                ->  Result  (cost=0.99..526919.26 rows=1173 width=1392) (actual time=0.155..3.982 rows=50 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=2279
                                      ->  Nested Loop Semi Join  (cost=0.99..526919.26 rows=1173 width=1384) (actual time=0.148..3.953 rows=50 loops=1)
                                            Buffers: shared hit=2279
                                            ->  Index Scan using records_list_updated on records r  (cost=0.42..501702.69 rows=5845 width=1384) (actual time=0.113..3.034 rows=117 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1670
                                                  Buffers: shared hit=1810
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..4.30 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=117)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=469
                          ->  Limit  (cost=0.99..1189.38 rows=27 width=1392) (actual time=0.048..0.092 rows=2 loops=1)
                                Buffers: shared hit=24
                                ->  Result  (cost=0.99..1189.38 rows=27 width=1392) (actual time=0.048..0.090 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=24
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.38 rows=27 width=1384) (actual time=0.045..0.086 rows=2 loops=1)
                                            Buffers: shared hit=24
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.023..0.034 rows=4 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Buffers: shared hit=7
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (actual time=0.011..0.011 rows=0 loops=4)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=17
                          ->  Limit  (cost=0.99..111.69 rows=2 width=1392) (actual time=0.083..0.086 rows=1 loops=1)
                                Buffers: shared hit=19
                                ->  Result  (cost=0.99..111.69 rows=2 width=1392) (actual time=0.083..0.085 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=19
                                      ->  Nested Loop Semi Join  (cost=0.99..111.69 rows=2 width=1384) (actual time=0.080..0.082 rows=1 loops=1)
                                            Buffers: shared hit=19
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.024..0.032 rows=3 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Buffers: shared hit=6
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.015..0.015 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=13
                          ->  Limit  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.187..0.195 rows=1 loops=1)
                                Buffers: shared hit=38
                                ->  Sort  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.187..0.193 rows=1 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=38
                                      ->  Result  (cost=36.98..45.96 rows=1 width=1392) (actual time=0.146..0.175 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=38
                                            ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=1384) (actual time=0.140..0.167 rows=1 loops=1)
                                                  Buffers: shared hit=38
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.086..0.116 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.062..0.065 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.046..0.055 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.035..0.036 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.018..0.019 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.015..0.015 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.015..0.016 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=13
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 8.729 ms
Execution Time: 5.307 ms
```

### bool eq 40%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=7270.54..7270.55 rows=1 width=8) (actual time=14.421..14.424 rows=1 loops=1)
  Buffers: shared hit=11764
  ->  Limit  (cost=0.99..7258.03 rows=1001 width=4) (actual time=0.247..14.248 rows=1001 loops=1)
        Buffers: shared hit=11764
        ->  Result  (cost=0.99..288296.15 rows=39766 width=4) (actual time=0.245..14.095 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=11764
              ->  Nested Loop Semi Join  (cost=0.99..288296.15 rows=39766 width=0) (actual time=0.229..13.892 rows=1001 loops=1)
                    Buffers: shared hit=11764
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.036..0.971 rows=2529 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=1647
                    ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..1.38 rows=1 width=32) (actual time=0.005..0.005 rows=0 loops=2529)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r.id))
                          Heap Fetches: 0
                          Buffers: shared hit=10117
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.330 ms
Execution Time: 14.561 ms
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
Limit  (cost=1.55..1011.52 rows=51 width=1426) (actual time=0.230..1.594 rows=51 loops=1)
  Buffers: shared hit=817
  ->  Result  (cost=1.55..787495.22 rows=39766 width=1426) (actual time=0.228..1.581 rows=51 loops=1)
        One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
        Buffers: shared hit=817
        ->  Nested Loop Left Join  (cost=1.55..787196.98 rows=39766 width=1399) (actual time=0.207..1.522 rows=51 loops=1)
              Buffers: shared hit=817
              ->  Nested Loop  (cost=1.25..774187.55 rows=39766 width=1391) (actual time=0.189..1.277 rows=51 loops=1)
                    Buffers: shared hit=664
                    ->  Nested Loop Semi Join  (cost=0.98..773682.18 rows=39766 width=1384) (actual time=0.170..1.221 rows=51 loops=1)
                          Buffers: shared hit=661
                          ->  Index Scan using records_list_updated on records r  (cost=0.42..500712.14 rows=198111 width=1384) (actual time=0.037..0.319 rows=131 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                                Buffers: shared hit=136
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..1.38 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=131)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6a4b171f-8fa5-4094-8761-66bc8da0526a'::uuid) AND (value = true) AND (record_id = r.id))
                                Heap Fetches: 0
                                Buffers: shared hit=525
                    ->  Materialize  (cost=0.27..8.30 rows=1 width=39) (actual time=0.000..0.001 rows=1 loops=51)
                          Buffers: shared hit=3
                          ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.011..0.012 rows=1 loops=1)
                                Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3
              ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=51)
                    One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                    Buffers: shared hit=153
                    ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=51)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                          Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.202 ms
Execution Time: 1.811 ms
```

### bool eq 2%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=2530.64..2530.65 rows=1 width=8) (actual time=26.802..26.814 rows=1 loops=1)
  Buffers: shared hit=16756
  ->  Limit  (cost=677.78..2529.87 rows=62 width=4) (actual time=6.271..26.706 rows=524 loops=1)
        Buffers: shared hit=16756
        ->  Result  (cost=677.78..2529.87 rows=62 width=4) (actual time=6.269..26.600 rows=524 loops=1)
              Buffers: shared hit=16756
              ->  Append  (cost=677.78..2529.25 rows=62 width=0) (actual time=6.267..26.457 rows=524 loops=1)
                    Buffers: shared hit=16756
                    ->  Subquery Scan on "*SELECT* 1"  (cost=677.78..1210.35 rows=59 width=0) (actual time=6.265..21.933 rows=512 loops=1)
                          Buffers: shared hit=15784
                          ->  Limit  (cost=677.78..1209.76 rows=59 width=4) (actual time=6.263..21.786 rows=512 loops=1)
                                Buffers: shared hit=15784
                                ->  Result  (cost=677.78..1209.76 rows=59 width=4) (actual time=6.262..21.685 rows=512 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15784
                                      ->  Hash Semi Join  (cost=677.78..1209.76 rows=59 width=0) (actual time=6.245..21.551 rows=512 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15784
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.052..9.737 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=576.98..576.98 rows=8019 width=32) (actual time=6.075..6.076 rows=8010 loops=1)
                                                  Buckets: 8192  Batches: 1  Memory Usage: 565kB
                                                  Buffers: shared hit=103
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..576.98 rows=8019 width=32) (actual time=0.047..3.136 rows=8010 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=103
                    ->  Subquery Scan on "*SELECT* 2"  (cost=569.70..1160.92 rows=1 width=0) (actual time=1.924..3.945 rows=12 loops=1)
                          Buffers: shared hit=624
                          ->  Limit  (cost=569.70..1160.91 rows=1 width=4) (actual time=1.922..3.940 rows=12 loops=1)
                                Buffers: shared hit=624
                                ->  Result  (cost=569.70..1160.91 rows=1 width=4) (actual time=1.919..3.935 rows=12 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=624
                                      ->  Merge Semi Join  (cost=569.70..1160.91 rows=1 width=0) (actual time=1.902..3.914 rows=12 loops=1)
                                            Merge Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=624
                                            ->  Sort  (cost=566.16..566.49 rows=135 width=32) (actual time=1.451..1.491 rows=484 loops=1)
                                                  Sort Key: r_1.id
                                                  Sort Method: quicksort  Memory: 51kB
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.308..1.257 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.192..0.192 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..576.98 rows=8019 width=32) (actual time=0.086..1.564 rows=8010 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=103
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.70 rows=1 width=0) (actual time=0.352..0.353 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..111.69 rows=1 width=4) (actual time=0.351..0.352 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.69 rows=1 width=4) (actual time=0.350..0.351 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.69 rows=1 width=0) (actual time=0.340..0.340 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.027..0.130 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.97 rows=1 width=0) (actual time=0.099..0.102 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.98..45.96 rows=1 width=4) (actual time=0.098..0.101 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.98..45.96 rows=1 width=4) (actual time=0.098..0.100 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=0) (actual time=0.095..0.097 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.060..0.076 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.043..0.045 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.031..0.036 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.022..0.023 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.009..0.009 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.012..0.013 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.009..0.009 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.825 ms
Execution Time: 27.204 ms
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
Nested Loop Left Join  (cost=21021.54..22266.56 rows=1 width=1434) (actual time=50.849..51.194 rows=51 loops=1)
  Buffers: shared hit=22771
  ->  Nested Loop  (cost=21021.24..22258.22 rows=1 width=1399) (actual time=50.836..51.087 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=22618
        ->  Subquery Scan on x  (cost=21020.96..22249.91 rows=1 width=1392) (actual time=50.825..51.007 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=22465
              ->  Limit  (cost=21020.96..22248.76 rows=51 width=1392) (actual time=50.822..50.979 rows=51 loops=1)
                    Buffers: shared hit=22465
                    ->  Merge Append  (cost=21020.96..22320.99 rows=54 width=1392) (actual time=50.820..50.973 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=22465
                          ->  Limit  (cost=20972.97..20973.10 rows=51 width=1392) (actual time=50.336..50.352 rows=50 loops=1)
                                Buffers: shared hit=21901
                                ->  Sort  (cost=20972.97..20973.12 rows=59 width=1392) (actual time=50.335..50.347 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: top-N heapsort  Memory: 183kB
                                      Buffers: shared hit=21901
                                      ->  Result  (cost=1136.55..20971.23 rows=59 width=1392) (actual time=19.822..49.954 rows=512 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=21901
                                            ->  Hash Semi Join  (cost=1136.55..20971.23 rows=59 width=1384) (actual time=19.810..49.837 rows=512 loops=1)
                                                  Hash Cond: (r.id = v.record_id)
                                                  Buffers: shared hit=21901
                                                  ->  Bitmap Heap Scan on records r  (cost=459.32..20277.78 rows=5845 width=1384) (actual time=16.310..42.610 rows=24118 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Heap Blocks: exact=21422
                                                        Buffers: shared hit=21797
                                                        ->  Bitmap Index Scan on records_list_group  (cost=0.00..457.86 rows=5845 width=0) (actual time=9.610..9.610 rows=24118 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Buffers: shared hit=375
                                                  ->  Hash  (cost=576.98..576.98 rows=8019 width=32) (actual time=3.364..3.365 rows=8010 loops=1)
                                                        Buckets: 8192  Batches: 1  Memory Usage: 565kB
                                                        Buffers: shared hit=104
                                                        ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..576.98 rows=8019 width=32) (actual time=0.037..1.768 rows=8010 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=104
                          ->  Limit  (cost=0.99..1189.38 rows=1 width=1392) (actual time=0.134..0.251 rows=2 loops=1)
                                Buffers: shared hit=216
                                ->  Result  (cost=0.99..1189.38 rows=1 width=1392) (actual time=0.133..0.249 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=216
                                      ->  Nested Loop Semi Join  (cost=0.99..1189.38 rows=1 width=1384) (actual time=0.122..0.237 rows=2 loops=1)
                                            Buffers: shared hit=216
                                            ->  Index Scan using records_list_requester on records r_1  (cost=0.42..569.72 rows=135 width=1384) (actual time=0.034..0.091 rows=42 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=47
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..4.58 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=42)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true) AND (record_id = r_1.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=169
                          ->  Limit  (cost=0.99..111.69 rows=1 width=1392) (actual time=0.250..0.251 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.69 rows=1 width=1392) (actual time=0.249..0.250 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.69 rows=1 width=1384) (actual time=0.246..0.247 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.011..0.084 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.002..0.002 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.095..0.097 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.094..0.097 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.98..45.96 rows=1 width=1392) (actual time=0.086..0.088 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=1384) (actual time=0.083..0.085 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.055..0.068 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.031..0.033 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.022..0.026 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.017..0.017 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.007..0.007 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.009..0.009 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.011..0.011 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.004..0.004 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.001..0.001 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 15.435 ms
Execution Time: 51.740 ms
```

### bool eq 2%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=7893.64..7893.65 rows=1 width=8) (actual time=22.595..22.600 rows=1 loops=1)
  Buffers: shared hit=32610
  ->  Limit  (cost=677.66..7881.13 rows=1001 width=4) (actual time=3.041..22.488 rows=1001 loops=1)
        Buffers: shared hit=32610
        ->  Result  (cost=677.66..14969.46 rows=1986 width=4) (actual time=3.039..22.364 rows=1001 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32610
              ->  Hash Semi Join  (cost=677.66..14969.46 rows=1986 width=0) (actual time=3.032..22.210 rows=1001 loops=1)
                    Hash Cond: (r.id = v.record_id)
                    Buffers: shared hit=32610
                    ->  Index Only Scan using records_list_updated on records r  (cost=0.42..13742.65 rows=198111 width=32) (actual time=0.023..12.312 rows=50170 loops=1)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid))
                          Heap Fetches: 0
                          Buffers: shared hit=32507
                    ->  Hash  (cost=576.98..576.98 rows=8019 width=32) (actual time=2.982..2.984 rows=8010 loops=1)
                          Buckets: 8192  Batches: 1  Memory Usage: 565kB
                          Buffers: shared hit=103
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..576.98 rows=8019 width=32) (actual time=0.022..1.602 rows=8010 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true))
                                Heap Fetches: 0
                                Buffers: shared hit=103
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 0.798 ms
Execution Time: 22.646 ms
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
Limit  (cost=1023.53..1023.65 rows=51 width=1426) (actual time=58.845..58.866 rows=51 loops=1)
  Buffers: shared hit=32201
  ->  Sort  (cost=1023.53..1028.49 rows=1986 width=1426) (actual time=58.843..58.860 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 197kB
        Buffers: shared hit=32201
        ->  Result  (cost=762.48..957.27 rows=1986 width=1426) (actual time=3.688..53.363 rows=8010 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=32201
              ->  Hash Left Join  (cost=762.48..942.37 rows=1986 width=1399) (actual time=3.676..49.009 rows=8010 loops=1)
                    Hash Cond: (r.assignment_group_id = g.id)
                    Buffers: shared hit=32201
                    ->  Nested Loop  (cost=617.77..792.46 rows=1986 width=1391) (actual time=3.541..45.888 rows=8010 loops=1)
                          Buffers: shared hit=32147
                          ->  Nested Loop  (cost=617.35..785.15 rows=1 width=71) (actual time=3.520..6.932 rows=8010 loops=1)
                                Buffers: shared hit=107
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.018..0.021 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=617.08..696.97 rows=7989 width=32) (actual time=3.500..5.618 rows=8010 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 1169kB
                                      Buffers: shared hit=104
                                      ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..576.98 rows=8019 width=32) (actual time=0.022..1.293 rows=8010 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '33c8b8da-79a2-4dd6-8598-395b8d528673'::uuid) AND (value = true))
                                            Heap Fetches: 0
                                            Buffers: shared hit=104
                          ->  Index Scan using records_pkey on records r  (cost=0.42..6.81 rows=1 width=1384) (actual time=0.004..0.004 rows=1 loops=8010)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=32040
                    ->  Hash  (cost=144.03..144.03 rows=53 width=40) (actual time=0.128..0.129 rows=53 loops=1)
                          Buckets: 1024  Batches: 1  Memory Usage: 12kB
                          Buffers: shared hit=54
                          ->  Result  (cost=4.71..144.03 rows=53 width=40) (actual time=0.028..0.118 rows=53 loops=1)
                                One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                Buffers: shared hit=54
                                ->  Bitmap Heap Scan on groups g  (cost=4.71..144.03 rows=53 width=40) (actual time=0.023..0.105 rows=53 loops=1)
                                      Recheck Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Heap Blocks: exact=52
                                      Buffers: shared hit=54
                                      ->  Bitmap Index Scan on groups_live_key  (cost=0.00..4.68 rows=53 width=0) (actual time=0.011..0.011 rows=53 loops=1)
                                            Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=2
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.549 ms
Execution Time: 58.995 ms
```

### bool eq 0.25%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=1412.07..1412.08 rows=1 width=8) (actual time=14.293..14.306 rows=1 loops=1)
  Buffers: shared hit=16582
  ->  Limit  (cost=80.83..1411.94 rows=10 width=4) (actual time=0.632..14.285 rows=64 loops=1)
        Buffers: shared hit=16582
        ->  Result  (cost=80.83..1411.94 rows=10 width=4) (actual time=0.630..14.270 rows=64 loops=1)
              Buffers: shared hit=16582
              ->  Append  (cost=80.83..1411.84 rows=10 width=0) (actual time=0.629..14.252 rows=64 loops=1)
                    Buffers: shared hit=16582
                    ->  Subquery Scan on "*SELECT* 1"  (cost=80.83..612.10 rows=7 width=0) (actual time=0.628..11.754 rows=63 loops=1)
                          Buffers: shared hit=15697
                          ->  Limit  (cost=80.83..612.03 rows=7 width=4) (actual time=0.627..11.737 rows=63 loops=1)
                                Buffers: shared hit=15697
                                ->  Result  (cost=80.83..612.03 rows=7 width=4) (actual time=0.626..11.725 rows=63 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=15697
                                      ->  Hash Semi Join  (cost=80.83..612.03 rows=7 width=0) (actual time=0.615..11.697 rows=63 loops=1)
                                            Hash Cond: (r.id = v.record_id)
                                            Buffers: shared hit=15697
                                            ->  Index Only Scan using records_list_group on records r  (cost=0.55..516.31 rows=5845 width=32) (actual time=0.046..7.631 rows=24118 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id IS NOT NULL) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=15681
                                            ->  Hash  (cost=68.94..68.94 rows=906 width=32) (actual time=0.525..0.526 rows=920 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 66kB
                                                  Buffers: shared hit=16
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..68.94 rows=906 width=32) (actual time=0.037..0.276 rows=920 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=16
                    ->  Subquery Scan on "*SELECT* 2"  (cost=94.82..642.02 rows=1 width=0) (actual time=1.714..1.990 rows=1 loops=1)
                          Buffers: shared hit=537
                          ->  Limit  (cost=94.82..642.01 rows=1 width=4) (actual time=1.713..1.987 rows=1 loops=1)
                                Buffers: shared hit=537
                                ->  Result  (cost=94.82..642.01 rows=1 width=4) (actual time=1.710..1.985 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=537
                                      ->  Hash Semi Join  (cost=94.82..642.01 rows=1 width=0) (actual time=1.694..1.967 rows=1 loops=1)
                                            Hash Cond: (r_1.id = v_1.record_id)
                                            Buffers: shared hit=537
                                            ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=32) (actual time=0.349..1.272 rows=484 loops=1)
                                                  Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 29
                                                  Heap Blocks: exact=511
                                                  Buffers: shared hit=521
                                                  ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.220..0.220 rows=513 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Buffers: shared hit=10
                                            ->  Hash  (cost=68.94..68.94 rows=906 width=32) (actual time=0.596..0.597 rows=920 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 66kB
                                                  Buffers: shared hit=16
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..68.94 rows=906 width=32) (actual time=0.051..0.308 rows=920 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=16
                    ->  Subquery Scan on "*SELECT* 3"  (cost=0.99..111.70 rows=1 width=0) (actual time=0.368..0.369 rows=0 loops=1)
                          Buffers: shared hit=311
                          ->  Limit  (cost=0.99..111.69 rows=1 width=4) (actual time=0.367..0.368 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.69 rows=1 width=4) (actual time=0.365..0.366 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.69 rows=1 width=0) (actual time=0.353..0.353 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.032..0.132 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.97 rows=1 width=0) (actual time=0.115..0.118 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.98..45.96 rows=1 width=4) (actual time=0.114..0.117 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.98..45.96 rows=1 width=4) (actual time=0.113..0.116 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=0) (actual time=0.110..0.112 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.072..0.090 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.052..0.054 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.039..0.044 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.016..0.016 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.266 ms
Execution Time: 14.636 ms
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
Nested Loop Left Join  (cost=8185.59..8312.75 rows=1 width=1434) (actual time=8.871..9.240 rows=51 loops=1)
  Buffers: shared hit=4887
  ->  Nested Loop  (cost=8185.29..8304.42 rows=1 width=1399) (actual time=8.847..9.061 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=4734
        ->  Subquery Scan on x  (cost=8185.01..8296.11 rows=1 width=1392) (actual time=8.831..8.925 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=4581
              ->  Limit  (cost=8185.01..8295.88 rows=10 width=1392) (actual time=8.828..8.881 rows=51 loops=1)
                    Buffers: shared hit=4581
                    ->  Merge Append  (cost=8185.01..8295.88 rows=10 width=1392) (actual time=8.826..8.872 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=4581
                          ->  Limit  (cost=7495.99..7496.00 rows=7 width=1392) (actual time=6.675..6.694 rows=50 loops=1)
                                Buffers: shared hit=3696
                                ->  Sort  (cost=7495.99..7496.00 rows=7 width=1392) (actual time=6.674..6.686 rows=50 loops=1)
                                      Sort Key: r.updated_at DESC, r.id DESC
                                      Sort Method: quicksort  Memory: 151kB
                                      Buffers: shared hit=3696
                                      ->  Result  (cost=73.91..7495.89 rows=7 width=1392) (actual time=0.958..6.503 rows=63 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=3696
                                            ->  Nested Loop  (cost=73.91..7495.89 rows=7 width=1384) (actual time=0.951..6.473 rows=63 loops=1)
                                                  Buffers: shared hit=3696
                                                  ->  HashAggregate  (cost=73.47..82.53 rows=906 width=32) (actual time=0.642..0.827 rows=920 loops=1)
                                                        Group Key: v.tenant_id, v.record_id
                                                        Batches: 1  Memory Usage: 169kB
                                                        Buffers: shared hit=16
                                                        ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..68.94 rows=906 width=32) (actual time=0.043..0.234 rows=920 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=16
                                                  ->  Index Scan using records_pkey on records r  (cost=0.42..8.18 rows=1 width=1384) (actual time=0.006..0.006 rows=0 loops=920)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                        Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 1
                                                        Buffers: shared hit=3680
                          ->  Limit  (cost=642.02..642.03 rows=1 width=1392) (actual time=1.688..1.692 rows=1 loops=1)
                                Buffers: shared hit=537
                                ->  Sort  (cost=642.02..642.03 rows=1 width=1392) (actual time=1.685..1.689 rows=1 loops=1)
                                      Sort Key: r_1.updated_at DESC, r_1.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=537
                                      ->  Result  (cost=94.82..642.01 rows=1 width=1392) (actual time=1.401..1.676 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=537
                                            ->  Hash Semi Join  (cost=94.82..642.01 rows=1 width=1384) (actual time=1.391..1.666 rows=1 loops=1)
                                                  Hash Cond: (r_1.id = v_1.record_id)
                                                  Buffers: shared hit=537
                                                  ->  Bitmap Heap Scan on records r_1  (cost=14.54..561.38 rows=135 width=1384) (actual time=0.251..1.133 rows=484 loops=1)
                                                        Recheck Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                        Filter: ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                        Rows Removed by Filter: 29
                                                        Heap Blocks: exact=511
                                                        Buffers: shared hit=521
                                                        ->  Bitmap Index Scan on records_list_requester  (cost=0.00..14.51 rows=139 width=0) (actual time=0.155..0.155 rows=513 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id IS NOT NULL) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Buffers: shared hit=10
                                                  ->  Hash  (cost=68.94..68.94 rows=906 width=32) (actual time=0.448..0.450 rows=920 loops=1)
                                                        Buckets: 1024  Batches: 1  Memory Usage: 66kB
                                                        Buffers: shared hit=16
                                                        ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v_1  (cost=0.56..68.94 rows=906 width=32) (actual time=0.042..0.241 rows=920 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=16
                          ->  Limit  (cost=0.99..111.69 rows=1 width=1392) (actual time=0.343..0.345 rows=0 loops=1)
                                Buffers: shared hit=311
                                ->  Result  (cost=0.99..111.69 rows=1 width=1392) (actual time=0.342..0.343 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=311
                                      ->  Nested Loop Semi Join  (cost=0.99..111.69 rows=1 width=1384) (actual time=0.335..0.336 rows=0 loops=1)
                                            Buffers: shared hit=311
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.027..0.126 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..4.58 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=61)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true) AND (record_id = r_2.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=244
                          ->  Limit  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.114..0.117 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Sort  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.113..0.116 rows=0 loops=1)
                                      Sort Key: r_3.updated_at DESC, r_3.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=37
                                      ->  Result  (cost=36.98..45.96 rows=1 width=1392) (actual time=0.106..0.109 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                            Buffers: shared hit=37
                                            ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=1384) (actual time=0.104..0.106 rows=0 loops=1)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.065..0.082 rows=3 loops=1)
                                                        Buffers: shared hit=25
                                                        ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.046..0.048 rows=3 loops=1)
                                                              Group Key: acl_share.tenant_id, acl_share.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=13
                                                              ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.035..0.040 rows=3 loops=1)
                                                                    Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Heap Blocks: exact=3
                                                                    Buffers: shared hit=13
                                                                    ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.028..0.029 rows=0 loops=1)
                                                                          Buffers: shared hit=10
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.013..0.013 rows=3 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                Buffers: shared hit=2
                                                                          ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                Buffers: shared hit=8
                                                        ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Buffers: shared hit=12
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.007..0.007 rows=0 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 10.460 ms
Execution Time: 9.657 ms
```

### bool eq 0.25%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=7492.15..7492.16 rows=1 width=8) (actual time=8.543..8.546 rows=1 loops=1)
  Buffers: shared hit=3696
  ->  Limit  (cost=73.91..7489.35 rows=224 width=4) (actual time=0.686..8.381 rows=920 loops=1)
        Buffers: shared hit=3696
        ->  Result  (cost=73.91..7489.35 rows=224 width=4) (actual time=0.685..8.171 rows=920 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=3696
              ->  Nested Loop  (cost=73.91..7489.35 rows=224 width=0) (actual time=0.676..7.969 rows=920 loops=1)
                    Buffers: shared hit=3696
                    ->  HashAggregate  (cost=73.47..82.53 rows=906 width=32) (actual time=0.656..0.911 rows=920 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 169kB
                          Buffers: shared hit=16
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..68.94 rows=906 width=32) (actual time=0.033..0.237 rows=920 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true))
                                Heap Fetches: 0
                                Buffers: shared hit=16
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.17 rows=1 width=32) (actual time=0.007..0.007 rows=1 loops=920)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=3680
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.049 ms
Execution Time: 8.616 ms
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
Limit  (cost=190.57..190.70 rows=51 width=1426) (actual time=12.509..12.527 rows=51 loops=1)
  Buffers: shared hit=6459
  ->  Sort  (cost=190.57..191.13 rows=224 width=1426) (actual time=12.507..12.517 rows=51 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: top-N heapsort  Memory: 199kB
        Buffers: shared hit=6459
        ->  Result  (cost=74.47..183.10 rows=224 width=1426) (actual time=0.646..11.629 rows=920 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=6459
              ->  Nested Loop Left Join  (cost=74.47..181.42 rows=224 width=1399) (actual time=0.630..10.889 rows=920 loops=1)
                    Buffers: shared hit=6459
                    ->  Nested Loop  (cost=74.17..108.13 rows=224 width=1391) (actual time=0.615..7.498 rows=920 loops=1)
                          Buffers: shared hit=3699
                          ->  Nested Loop  (cost=73.74..99.88 rows=1 width=71) (actual time=0.597..1.061 rows=920 loops=1)
                                Buffers: shared hit=19
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.023..0.026 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=73.47..82.53 rows=906 width=32) (actual time=0.573..0.806 rows=920 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 169kB
                                      Buffers: shared hit=16
                                      ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..68.94 rows=906 width=32) (actual time=0.026..0.215 rows=920 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '453e43c2-1bb8-4c95-91cf-c27cdb0049e3'::uuid) AND (value = true))
                                            Heap Fetches: 0
                                            Buffers: shared hit=16
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.17 rows=1 width=1384) (actual time=0.006..0.006 rows=1 loops=920)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=3680
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.003..0.003 rows=1 loops=920)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=2760
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.002..0.002 rows=1 loops=920)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=2760
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.987 ms
Execution Time: 12.630 ms
```

### bool eq 0.01%, ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM ((SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND (((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid)) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))) LIMIT 1001) UNION ALL (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $8::uuid AND v.value = $9::boolean) AND ((EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[]))))) AND NOT (true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[]))) AND NOT ((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid)) AND NOT ((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))) LIMIT 1001)) u LIMIT 1001) c
```

```
Aggregate  (cost=729.79..729.80 rows=1 width=8) (actual time=0.943..0.955 rows=1 loops=1)
  Buffers: shared hit=391
  ->  Limit  (cost=5.98..729.74 rows=4 width=4) (actual time=0.250..0.950 rows=2 loops=1)
        Buffers: shared hit=391
        ->  Result  (cost=5.98..729.74 rows=4 width=4) (actual time=0.249..0.947 rows=2 loops=1)
              Buffers: shared hit=391
              ->  Append  (cost=5.98..729.70 rows=4 width=0) (actual time=0.248..0.945 rows=2 loops=1)
                    Buffers: shared hit=391
                    ->  Subquery Scan on "*SELECT* 1"  (cost=5.98..310.58 rows=1 width=0) (actual time=0.247..0.418 rows=2 loops=1)
                          Buffers: shared hit=141
                          ->  Limit  (cost=5.98..310.57 rows=1 width=4) (actual time=0.246..0.416 rows=2 loops=1)
                                Buffers: shared hit=141
                                ->  Result  (cost=5.98..310.57 rows=1 width=4) (actual time=0.245..0.415 rows=2 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=141
                                      ->  Nested Loop  (cost=5.98..310.57 rows=1 width=0) (actual time=0.233..0.402 rows=2 loops=1)
                                            Buffers: shared hit=141
                                            ->  HashAggregate  (cost=5.54..5.91 rows=36 width=32) (actual time=0.072..0.081 rows=34 loops=1)
                                                  Group Key: v.tenant_id, v.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..5.37 rows=36 width=32) (actual time=0.047..0.055 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.009..0.009 rows=0 loops=34)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                  Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=136
                    ->  Subquery Scan on "*SELECT* 2"  (cost=5.98..310.67 rows=1 width=0) (actual time=0.163..0.166 rows=0 loops=1)
                          Buffers: shared hit=141
                          ->  Limit  (cost=5.98..310.66 rows=1 width=4) (actual time=0.163..0.165 rows=0 loops=1)
                                Buffers: shared hit=141
                                ->  Result  (cost=5.98..310.66 rows=1 width=4) (actual time=0.162..0.164 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=141
                                      ->  Nested Loop  (cost=5.98..310.66 rows=1 width=0) (actual time=0.158..0.159 rows=0 loops=1)
                                            Buffers: shared hit=141
                                            ->  HashAggregate  (cost=5.54..5.91 rows=36 width=32) (actual time=0.034..0.041 rows=34 loops=1)
                                                  Group Key: v_1.tenant_id, v_1.record_id
                                                  Batches: 1  Memory Usage: 24kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_1  (cost=0.56..5.37 rows=36 width=32) (actual time=0.016..0.021 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                                            ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=32) (actual time=0.003..0.003 rows=0 loops=34)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v_1.record_id))
                                                  Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 1
                                                  Buffers: shared hit=136
                    ->  Subquery Scan on "*SELECT* 3"  (cost=6.25..62.46 rows=1 width=0) (actual time=0.238..0.240 rows=0 loops=1)
                          Buffers: shared hit=72
                          ->  Limit  (cost=6.25..62.45 rows=1 width=4) (actual time=0.237..0.239 rows=0 loops=1)
                                Buffers: shared hit=72
                                ->  Result  (cost=6.25..62.45 rows=1 width=4) (actual time=0.237..0.238 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=72
                                      ->  Hash Semi Join  (cost=6.25..62.45 rows=1 width=0) (actual time=0.235..0.236 rows=0 loops=1)
                                            Hash Cond: (r_2.id = v_2.record_id)
                                            Buffers: shared hit=72
                                            ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=32) (actual time=0.022..0.152 rows=61 loops=1)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                  Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                  Rows Removed by Filter: 3
                                                  Buffers: shared hit=67
                                            ->  Hash  (cost=5.37..5.37 rows=36 width=32) (actual time=0.030..0.030 rows=34 loops=1)
                                                  Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                  Buffers: shared hit=5
                                                  ->  Index Only Scan using rfv_bool_value on rfv_bool v_2  (cost=0.56..5.37 rows=36 width=32) (actual time=0.013..0.019 rows=34 loops=1)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                                        Heap Fetches: 0
                                                        Buffers: shared hit=5
                    ->  Subquery Scan on "*SELECT* 4"  (cost=36.98..45.97 rows=1 width=0) (actual time=0.115..0.117 rows=0 loops=1)
                          Buffers: shared hit=37
                          ->  Limit  (cost=36.98..45.96 rows=1 width=4) (actual time=0.114..0.116 rows=0 loops=1)
                                Buffers: shared hit=37
                                ->  Result  (cost=36.98..45.96 rows=1 width=4) (actual time=0.113..0.115 rows=0 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Buffers: shared hit=37
                                      ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=0) (actual time=0.110..0.112 rows=0 loops=1)
                                            Buffers: shared hit=37
                                            ->  Nested Loop  (cost=36.41..44.45 rows=1 width=48) (actual time=0.073..0.092 rows=3 loops=1)
                                                  Buffers: shared hit=25
                                                  ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.055..0.058 rows=3 loops=1)
                                                        Group Key: acl_share.tenant_id, acl_share.record_id
                                                        Batches: 1  Memory Usage: 24kB
                                                        Buffers: shared hit=13
                                                        ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.046..0.051 rows=3 loops=1)
                                                              Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Heap Blocks: exact=3
                                                              Buffers: shared hit=13
                                                              ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.036..0.036 rows=0 loops=1)
                                                                    Buffers: shared hit=10
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.019..0.019 rows=3 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                          Buffers: shared hit=2
                                                                    ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.016..0.016 rows=0 loops=1)
                                                                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                          Buffers: shared hit=8
                                                  ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=32) (actual time=0.010..0.010 rows=1 loops=3)
                                                        Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                        Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                        Buffers: shared hit=12
                                            ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                  Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                  Heap Fetches: 0
                                                  Buffers: shared hit=12
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 6.107 ms
Execution Time: 1.208 ms
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
Nested Loop Left Join  (cost=730.34..746.50 rows=1 width=1434) (actual time=1.054..1.080 rows=2 loops=1)
  Buffers: shared hit=406
  ->  Nested Loop  (cost=730.04..738.17 rows=1 width=1399) (actual time=1.030..1.049 rows=2 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=400
        ->  Subquery Scan on x  (cost=729.76..729.86 rows=1 width=1392) (actual time=1.012..1.027 rows=2 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=394
              ->  Limit  (cost=729.76..729.77 rows=4 width=1392) (actual time=1.009..1.021 rows=2 loops=1)
                    Buffers: shared hit=394
                    ->  Sort  (cost=729.76..729.77 rows=4 width=1392) (actual time=1.008..1.019 rows=2 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Sort Method: quicksort  Memory: 29kB
                          Buffers: shared hit=394
                          ->  Append  (cost=310.58..729.72 rows=4 width=1392) (actual time=0.463..1.009 rows=2 loops=1)
                                Buffers: shared hit=394
                                ->  Limit  (cost=310.58..310.58 rows=1 width=1392) (actual time=0.461..0.465 rows=2 loops=1)
                                      Buffers: shared hit=142
                                      ->  Sort  (cost=310.58..310.58 rows=1 width=1392) (actual time=0.461..0.463 rows=2 loops=1)
                                            Sort Key: r.updated_at DESC, r.id DESC
                                            Sort Method: quicksort  Memory: 29kB
                                            Buffers: shared hit=142
                                            ->  Result  (cost=5.98..310.57 rows=1 width=1392) (actual time=0.280..0.452 rows=2 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=142
                                                  ->  Nested Loop  (cost=5.98..310.57 rows=1 width=1384) (actual time=0.273..0.444 rows=2 loops=1)
                                                        Buffers: shared hit=142
                                                        ->  HashAggregate  (cost=5.54..5.91 rows=36 width=32) (actual time=0.109..0.117 rows=34 loops=1)
                                                              Group Key: v.tenant_id, v.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=6
                                                              ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..5.37 rows=36 width=32) (actual time=0.080..0.091 rows=34 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=6
                                                        ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.009..0.009 rows=0 loops=34)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                                              Filter: ((assignment_group_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignment_group_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=136
                                ->  Limit  (cost=310.67..310.67 rows=1 width=1392) (actual time=0.196..0.197 rows=0 loops=1)
                                      Buffers: shared hit=142
                                      ->  Sort  (cost=310.67..310.67 rows=1 width=1392) (actual time=0.195..0.196 rows=0 loops=1)
                                            Sort Key: r_1.updated_at DESC, r_1.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=142
                                            ->  Result  (cost=5.98..310.66 rows=1 width=1392) (actual time=0.187..0.188 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=142
                                                  ->  Nested Loop  (cost=5.98..310.66 rows=1 width=1384) (actual time=0.182..0.183 rows=0 loops=1)
                                                        Buffers: shared hit=142
                                                        ->  HashAggregate  (cost=5.54..5.91 rows=36 width=32) (actual time=0.035..0.042 rows=34 loops=1)
                                                              Group Key: v_1.tenant_id, v_1.record_id
                                                              Batches: 1  Memory Usage: 24kB
                                                              Buffers: shared hit=6
                                                              ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v_1  (cost=0.56..5.37 rows=36 width=32) (actual time=0.016..0.021 rows=34 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=6
                                                        ->  Index Scan using records_pkey on records r_1  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.004..0.004 rows=0 loops=34)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v_1.record_id))
                                                              Filter: ((requester_id IS NOT NULL) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (requester_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 1
                                                              Buffers: shared hit=136
                                ->  Limit  (cost=62.46..62.47 rows=1 width=1392) (actual time=0.223..0.225 rows=0 loops=1)
                                      Buffers: shared hit=73
                                      ->  Sort  (cost=62.46..62.47 rows=1 width=1392) (actual time=0.223..0.225 rows=0 loops=1)
                                            Sort Key: r_2.updated_at DESC, r_2.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=73
                                            ->  Result  (cost=6.25..62.45 rows=1 width=1392) (actual time=0.218..0.220 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=73
                                                  ->  Hash Semi Join  (cost=6.25..62.45 rows=1 width=1384) (actual time=0.216..0.218 rows=0 loops=1)
                                                        Hash Cond: (r_2.id = v_2.record_id)
                                                        Buffers: shared hit=73
                                                        ->  Index Scan using records_list_assignee on records r_2  (cost=0.42..56.60 rows=12 width=1384) (actual time=0.022..0.160 rows=61 loops=1)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND (assignee_id IS NOT NULL) AND (assignee_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                              Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                              Rows Removed by Filter: 3
                                                              Buffers: shared hit=67
                                                        ->  Hash  (cost=5.37..5.37 rows=36 width=32) (actual time=0.036..0.036 rows=34 loops=1)
                                                              Buckets: 1024  Batches: 1  Memory Usage: 11kB
                                                              Buffers: shared hit=6
                                                              ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v_2  (cost=0.56..5.37 rows=36 width=32) (actual time=0.018..0.024 rows=34 loops=1)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                                                    Heap Fetches: 0
                                                                    Buffers: shared hit=6
                                ->  Limit  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.113..0.116 rows=0 loops=1)
                                      Buffers: shared hit=37
                                      ->  Sort  (cost=45.97..45.97 rows=1 width=1392) (actual time=0.113..0.116 rows=0 loops=1)
                                            Sort Key: r_3.updated_at DESC, r_3.id DESC
                                            Sort Method: quicksort  Memory: 25kB
                                            Buffers: shared hit=37
                                            ->  Result  (cost=36.98..45.96 rows=1 width=1392) (actual time=0.107..0.110 rows=0 loops=1)
                                                  One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                                  Buffers: shared hit=37
                                                  ->  Nested Loop Semi Join  (cost=36.98..45.96 rows=1 width=1384) (actual time=0.105..0.107 rows=0 loops=1)
                                                        Buffers: shared hit=37
                                                        ->  Nested Loop  (cost=36.41..44.45 rows=1 width=1400) (actual time=0.066..0.085 rows=3 loops=1)
                                                              Buffers: shared hit=25
                                                              ->  HashAggregate  (cost=35.98..35.99 rows=1 width=32) (actual time=0.050..0.052 rows=3 loops=1)
                                                                    Group Key: acl_share.tenant_id, acl_share.record_id
                                                                    Batches: 1  Memory Usage: 24kB
                                                                    Buffers: shared hit=13
                                                                    ->  Bitmap Heap Scan on record_shares acl_share  (cost=21.48..35.98 rows=1 width=32) (actual time=0.040..0.046 rows=3 loops=1)
                                                                          Recheck Cond: (((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Filter: (((principal_kind = 'user'::text) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) OR ((principal_kind = 'group'::text) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                          Heap Blocks: exact=3
                                                                          Buffers: shared hit=13
                                                                          ->  BitmapOr  (cost=21.48..21.48 rows=4 width=0) (actual time=0.031..0.032 rows=0 loops=1)
                                                                                Buffers: shared hit=10
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..4.29 rows=1 width=0) (actual time=0.015..0.016 rows=3 loops=1)
                                                                                      Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid))
                                                                                      Buffers: shared hit=2
                                                                                ->  Bitmap Index Scan on record_shares_principal  (cost=0.00..17.18 rows=3 width=0) (actual time=0.014..0.014 rows=0 loops=1)
                                                                                      Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (principal_id = ANY ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[])))
                                                                                      Buffers: shared hit=8
                                                              ->  Index Scan using records_pkey on records r_3  (cost=0.42..8.46 rows=1 width=1384) (actual time=0.009..0.009 rows=1 loops=3)
                                                                    Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = acl_share.record_id))
                                                                    Filter: (((requester_id IS NULL) OR (requester_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND ((assignee_id IS NULL) OR (assignee_id <> '59fb22d0-8db4-ebf3-0411-bc5e59f13344'::uuid)) AND (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid) AND ((assignment_group_id IS NULL) OR (assignment_group_id <> ALL ('{9e8a3462-a1b0-3ab1-93b3-a00e99528d30,969c31f2-d8fe-635a-5a2d-95d39e4e7489,2f067c2d-b39a-c447-6bd8-6f26d53e8105,e43d6642-9bbe-05be-b6a6-8d3a87bf4141}'::uuid[]))))
                                                                    Buffers: shared hit=12
                                                        ->  Index Only Scan using rfv_bool_value on rfv_bool v_3  (cost=0.56..1.48 rows=1 width=32) (actual time=0.006..0.006 rows=0 loops=3)
                                                              Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true) AND (record_id = r_3.id))
                                                              Heap Fetches: 0
                                                              Buffers: shared hit=12
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.007..0.007 rows=1 loops=2)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=6
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.008..0.008 rows=1 loops=2)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=6
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 8.492 ms
Execution Time: 1.465 ms
```

### bool eq 0.01%, no ACL

```sql
SELECT count(*) FROM (SELECT 1 FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND ((((r.requester_id IS NOT NULL AND r.requester_id = $3::uuid) OR (r.assignee_id IS NOT NULL AND r.assignee_id = $4::uuid) OR EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $5::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($6::uuid[])))))) OR ((true))) AND EXISTS (SELECT 1 FROM rfv_bool v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $7::uuid AND v.value = $8::boolean) LIMIT 1001) c
```

```
Aggregate  (cost=310.50..310.51 rows=1 width=8) (actual time=0.532..0.534 rows=1 loops=1)
  Buffers: shared hit=141
  ->  Limit  (cost=5.98..310.39 rows=9 width=4) (actual time=0.127..0.523 rows=34 loops=1)
        Buffers: shared hit=141
        ->  Result  (cost=5.98..310.39 rows=9 width=4) (actual time=0.126..0.513 rows=34 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=141
              ->  Nested Loop  (cost=5.98..310.39 rows=9 width=0) (actual time=0.112..0.491 rows=34 loops=1)
                    Buffers: shared hit=141
                    ->  HashAggregate  (cost=5.54..5.91 rows=36 width=32) (actual time=0.082..0.094 rows=34 loops=1)
                          Group Key: v.tenant_id, v.record_id
                          Batches: 1  Memory Usage: 24kB
                          Buffers: shared hit=5
                          ->  Index Only Scan using rfv_bool_value on rfv_bool v  (cost=0.56..5.37 rows=36 width=32) (actual time=0.050..0.058 rows=34 loops=1)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                Heap Fetches: 0
                                Buffers: shared hit=5
                    ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=32) (actual time=0.011..0.011 rows=1 loops=34)
                          Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                          Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                          Buffers: shared hit=136
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 1.578 ms
Execution Time: 0.613 ms
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
Limit  (cost=26.17..26.20 rows=9 width=1426) (actual time=0.940..0.954 rows=34 loops=1)
  Buffers: shared hit=247
  ->  Sort  (cost=26.17..26.20 rows=9 width=1426) (actual time=0.938..0.946 rows=34 loops=1)
        Sort Key: r.updated_at DESC, r.id DESC
        Sort Method: quicksort  Memory: 93kB
        Buffers: shared hit=247
        ->  Result  (cost=6.54..26.03 rows=9 width=1426) (actual time=0.181..0.869 rows=34 loops=1)
              One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
              Buffers: shared hit=247
              ->  Nested Loop Left Join  (cost=6.54..25.96 rows=9 width=1399) (actual time=0.157..0.807 rows=34 loops=1)
                    Buffers: shared hit=247
                    ->  Nested Loop  (cost=6.24..23.01 rows=9 width=1391) (actual time=0.136..0.572 rows=34 loops=1)
                          Buffers: shared hit=145
                          ->  Nested Loop  (cost=5.82..14.56 rows=1 width=71) (actual time=0.110..0.138 rows=34 loops=1)
                                Buffers: shared hit=9
                                ->  Index Scan using entity_definitions_number_prefix_key on entity_definitions d  (cost=0.27..8.29 rows=1 width=39) (actual time=0.032..0.035 rows=1 loops=1)
                                      Index Cond: (tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                                      Filter: (id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                      Buffers: shared hit=3
                                ->  HashAggregate  (cost=5.54..5.91 rows=36 width=32) (actual time=0.075..0.089 rows=34 loops=1)
                                      Group Key: v.tenant_id, v.record_id
                                      Batches: 1  Memory Usage: 24kB
                                      Buffers: shared hit=6
                                      ->  Index Only Scan Backward using rfv_bool_value on rfv_bool v  (cost=0.56..5.37 rows=36 width=32) (actual time=0.037..0.050 rows=34 loops=1)
                                            Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (field_id = '6d7c9876-6561-4e38-8f83-df14b58e78b1'::uuid) AND (value = true))
                                            Heap Fetches: 0
                                            Buffers: shared hit=6
                          ->  Index Scan using records_pkey on records r  (cost=0.42..8.45 rows=1 width=1384) (actual time=0.012..0.012 rows=1 loops=34)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = v.record_id))
                                Filter: (entity_definition_id = 'c856de3b-d7d1-4af1-b029-f8a365134bec'::uuid)
                                Buffers: shared hit=136
                    ->  Result  (cost=0.29..0.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=34)
                          One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid)
                          Buffers: shared hit=102
                          ->  Index Scan using groups_pkey on groups g  (cost=0.29..0.32 rows=1 width=40) (actual time=0.004..0.004 rows=1 loops=34)
                                Index Cond: ((tenant_id = 'a6aff943-dbce-4472-b77b-1b24e7b70819'::uuid) AND (id = r.assignment_group_id))
                                Buffers: shared hit=102
Settings: effective_cache_size = '6GB', plan_cache_mode = 'force_custom_plan', jit = 'off'
Planning Time: 2.970 ms
Execution Time: 1.124 ms
```
