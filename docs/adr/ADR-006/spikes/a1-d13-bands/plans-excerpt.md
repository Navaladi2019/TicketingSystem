# Plans for the two D13 bands A1 re-budgets (excerpt of tick992-explain.md)

Source: PR #775, `docs/data/field-index-measured/tick992-explain.md` at the head of `TICK-992/idle-host-scoring` (lists at `ea1b1785`). Single client, `nava_app`, `force_custom_plan`. Copied verbatim; the full file has all 40 cells.

## 40% with a narrowing ACL: the capped count (num lt)

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


## 2% without an ACL: the page (num lt)

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

