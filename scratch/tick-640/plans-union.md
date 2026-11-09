## requester fccbb3fd-048d-a9ac-ec41-5c1548c480a0, page 1
```
Nested Loop Left Join  (cost=18.50..239.75 rows=1 width=1243) (actual time=0.347..1.700 rows=51 loops=1)
  Buffers: shared hit=369
  ->  Nested Loop  (cost=18.20..231.42 rows=1 width=1242) (actual time=0.311..1.293 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=216
        ->  Subquery Scan on x  (cost=17.92..223.11 rows=1 width=1235) (actual time=0.285..1.158 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=63
              ->  Limit  (cost=17.92..221.96 rows=51 width=1235) (actual time=0.283..1.116 rows=51 loops=1)
                    Buffers: shared hit=63
                    ->  Merge Append  (cost=17.92..281.98 rows=66 width=1235) (actual time=0.281..1.108 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=63
                          ->  Limit  (cost=0.55..204.16 rows=51 width=1235) (actual time=0.118..0.930 rows=51 loops=1)
                                Buffers: shared hit=56
                                ->  Result  (cost=0.55..5729.64 rows=1435 width=1235) (actual time=0.117..0.922 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=56
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..5729.64 rows=1435 width=1235) (actual time=0.112..0.900 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Buffers: shared hit=56
                          ->  Limit  (cost=0.55..60.14 rows=14 width=1235) (actual time=0.115..0.116 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..60.14 rows=14 width=1235) (actual time=0.115..0.115 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..60.14 rows=14 width=1235) (actual time=0.113..0.113 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                            Filter: (requester_id IS DISTINCT FROM 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=16.79..16.80 rows=1 width=1235) (actual time=0.047..0.050 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=16.79..16.80 rows=1 width=1235) (actual time=0.047..0.049 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=0.72..16.78 rows=1 width=1235) (actual time=0.035..0.036 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=0.72..16.78 rows=1 width=1235) (actual time=0.034..0.035 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  Index Scan using record_shares_principal on record_shares s  (cost=0.29..8.31 rows=1 width=32) (actual time=0.033..0.034 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                        Filter: (principal_kind = 'user'::text)
                                                        Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1235) (never executed)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = s.record_id))
                                                        Filter: ((requester_id IS DISTINCT FROM 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (assignee_id IS DISTINCT FROM 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.007..0.007 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=604
Planning Time: 2.837 ms
Execution Time: 1.962 ms
```
## requester fccbb3fd-048d-a9ac-ec41-5c1548c480a0, page 20 (cursor 2026-09-29 03:10:06.448328+00,0150dafd-b1c7-41f0-99ea-b49cdcff90ec)
```
Nested Loop Left Join  (cost=18.50..240.01 rows=1 width=1243) (actual time=0.375..1.726 rows=51 loops=1)
  Buffers: shared hit=369
  ->  Nested Loop  (cost=18.20..231.68 rows=1 width=1242) (actual time=0.343..1.338 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=216
        ->  Subquery Scan on x  (cost=17.92..223.37 rows=1 width=1235) (actual time=0.315..1.204 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=63
              ->  Limit  (cost=17.92..222.22 rows=51 width=1235) (actual time=0.312..1.160 rows=51 loops=1)
                    Buffers: shared hit=63
                    ->  Merge Append  (cost=17.92..274.29 rows=64 width=1235) (actual time=0.311..1.152 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=63
                          ->  Limit  (cost=0.55..204.42 rows=51 width=1235) (actual time=0.142..0.970 rows=51 loops=1)
                                Buffers: shared hit=56
                                ->  Result  (cost=0.55..4737.43 rows=1185 width=1235) (actual time=0.142..0.962 rows=51 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=56
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..4737.43 rows=1185 width=1235) (actual time=0.136..0.941 rows=51 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:10:06.448328+00'::timestamp with time zone, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid)))
                                            Buffers: shared hit=56
                          ->  Limit  (cost=0.55..52.23 rows=12 width=1235) (actual time=0.118..0.119 rows=1 loops=1)
                                Buffers: shared hit=5
                                ->  Result  (cost=0.55..52.23 rows=12 width=1235) (actual time=0.117..0.117 rows=1 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=5
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..52.23 rows=12 width=1235) (actual time=0.114..0.114 rows=1 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:10:06.448328+00'::timestamp with time zone, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid)))
                                            Filter: (requester_id IS DISTINCT FROM 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid)
                                            Buffers: shared hit=5
                          ->  Limit  (cost=16.80..16.80 rows=1 width=1235) (actual time=0.049..0.052 rows=0 loops=1)
                                Buffers: shared hit=2
                                ->  Sort  (cost=16.80..16.80 rows=1 width=1235) (actual time=0.049..0.051 rows=0 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 25kB
                                      Buffers: shared hit=2
                                      ->  Result  (cost=0.72..16.79 rows=1 width=1235) (actual time=0.036..0.038 rows=0 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=2
                                            ->  Nested Loop  (cost=0.72..16.79 rows=1 width=1235) (actual time=0.035..0.036 rows=0 loops=1)
                                                  Buffers: shared hit=2
                                                  ->  Index Scan using record_shares_principal on record_shares s  (cost=0.29..8.31 rows=1 width=32) (actual time=0.034..0.035 rows=0 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid))
                                                        Filter: (principal_kind = 'user'::text)
                                                        Buffers: shared hit=2
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1235) (never executed)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = s.record_id))
                                                        Filter: ((requester_id IS DISTINCT FROM 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (assignee_id IS DISTINCT FROM 'fccbb3fd-048d-a9ac-ec41-5c1548c480a0'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-09-29 03:10:06.448328+00'::timestamp with time zone, '0150dafd-b1c7-41f0-99ea-b49cdcff90ec'::uuid)))
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.001..0.001 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=593
Planning Time: 2.971 ms
Execution Time: 2.002 ms
```
## requester f222cde6-739e-cb4e-eaa6-14c1a2bed245, page 1
```
Nested Loop Left Join  (cost=18.50..240.88 rows=1 width=1243) (actual time=0.899..3.618 rows=51 loops=1)
  Buffers: shared hit=379
  ->  Nested Loop  (cost=18.20..232.56 rows=1 width=1242) (actual time=0.850..2.887 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=226
        ->  Subquery Scan on x  (cost=17.92..224.24 rows=1 width=1235) (actual time=0.807..2.639 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=73
              ->  Limit  (cost=17.92..223.10 rows=51 width=1235) (actual time=0.800..2.540 rows=51 loops=1)
                    Buffers: shared hit=73
                    ->  Merge Append  (cost=17.92..283.44 rows=66 width=1235) (actual time=0.796..2.526 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=73
                          ->  Limit  (cost=0.55..205.63 rows=51 width=1235) (actual time=0.173..1.750 rows=47 loops=1)
                                Buffers: shared hit=52
                                ->  Result  (cost=0.55..1415.99 rows=352 width=1235) (actual time=0.172..1.739 rows=47 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=52
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..1415.99 rows=352 width=1235) (actual time=0.158..1.703 rows=47 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                            Buffers: shared hit=52
                          ->  Limit  (cost=0.55..60.14 rows=14 width=1235) (actual time=0.200..0.325 rows=5 loops=1)
                                Buffers: shared hit=9
                                ->  Result  (cost=0.55..60.14 rows=14 width=1235) (actual time=0.198..0.322 rows=5 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=9
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..60.14 rows=14 width=1235) (actual time=0.192..0.313 rows=5 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                            Filter: (requester_id IS DISTINCT FROM 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)
                                            Buffers: shared hit=9
                          ->  Limit  (cost=16.79..16.80 rows=1 width=1235) (actual time=0.416..0.420 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Sort  (cost=16.79..16.80 rows=1 width=1235) (actual time=0.415..0.418 rows=1 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 29kB
                                      Buffers: shared hit=12
                                      ->  Result  (cost=0.72..16.78 rows=1 width=1235) (actual time=0.219..0.382 rows=2 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=12
                                            ->  Nested Loop  (cost=0.72..16.78 rows=1 width=1235) (actual time=0.213..0.375 rows=2 loops=1)
                                                  Buffers: shared hit=12
                                                  ->  Index Scan using record_shares_principal on record_shares s  (cost=0.29..8.31 rows=1 width=32) (actual time=0.068..0.123 rows=2 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                                        Filter: (principal_kind = 'user'::text)
                                                        Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.45 rows=1 width=1235) (actual time=0.120..0.121 rows=1 loops=2)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = s.record_id))
                                                        Filter: ((requester_id IS DISTINCT FROM 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) AND (assignee_id IS DISTINCT FROM 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid))
                                                        Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.003..0.003 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.012..0.012 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=604
Planning Time: 5.555 ms
Execution Time: 4.009 ms
```
## requester f222cde6-739e-cb4e-eaa6-14c1a2bed245, page 20 (cursor 2026-02-15 00:27:00.844898+00,09b90e91-b333-8cdc-5df8-c0dac0aaa909)
```
Nested Loop Left Join  (cost=18.50..242.59 rows=1 width=1243) (actual time=0.747..2.535 rows=51 loops=1)
  Buffers: shared hit=379
  ->  Nested Loop  (cost=18.20..234.26 rows=1 width=1242) (actual time=0.691..2.115 rows=51 loops=1)
        Join Filter: (d.id = x.entity_definition_id)
        Buffers: shared hit=226
        ->  Subquery Scan on x  (cost=17.92..225.95 rows=1 width=1235) (actual time=0.658..1.884 rows=51 loops=1)
              Filter: (x.tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=73
              ->  Limit  (cost=17.92..224.80 rows=51 width=1235) (actual time=0.653..1.814 rows=51 loops=1)
                    Buffers: shared hit=73
                    ->  Merge Append  (cost=17.92..236.97 rows=54 width=1235) (actual time=0.652..1.803 rows=51 loops=1)
                          Sort Key: r.updated_at DESC, r.id DESC
                          Buffers: shared hit=73
                          ->  Limit  (cost=0.55..206.90 rows=51 width=1235) (actual time=0.133..1.185 rows=47 loops=1)
                                Buffers: shared hit=52
                                ->  Result  (cost=0.55..206.90 rows=51 width=1235) (actual time=0.133..1.175 rows=47 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=52
                                      ->  Index Scan using records_list_requester on records r  (cost=0.55..206.90 rows=51 width=1235) (actual time=0.128..1.150 rows=47 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (requester_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) AND (ROW(updated_at, id) < ROW('2026-02-15 00:27:00.844898+00'::timestamp with time zone, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid)))
                                            Buffers: shared hit=52
                          ->  Limit  (cost=0.55..12.54 rows=2 width=1235) (actual time=0.173..0.252 rows=5 loops=1)
                                Buffers: shared hit=9
                                ->  Result  (cost=0.55..12.54 rows=2 width=1235) (actual time=0.172..0.250 rows=5 loops=1)
                                      One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                      Buffers: shared hit=9
                                      ->  Index Scan using records_list_assignee on records r_1  (cost=0.55..12.54 rows=2 width=1235) (actual time=0.170..0.245 rows=5 loops=1)
                                            Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (assignee_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) AND (ROW(updated_at, id) < ROW('2026-02-15 00:27:00.844898+00'::timestamp with time zone, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid)))
                                            Filter: (requester_id IS DISTINCT FROM 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid)
                                            Buffers: shared hit=9
                          ->  Limit  (cost=16.80..16.80 rows=1 width=1235) (actual time=0.342..0.344 rows=1 loops=1)
                                Buffers: shared hit=12
                                ->  Sort  (cost=16.80..16.80 rows=1 width=1235) (actual time=0.341..0.343 rows=1 loops=1)
                                      Sort Key: r_2.updated_at DESC, r_2.id DESC
                                      Sort Method: quicksort  Memory: 27kB
                                      Buffers: shared hit=12
                                      ->  Result  (cost=0.72..16.79 rows=1 width=1235) (actual time=0.225..0.322 rows=1 loops=1)
                                            One-Time Filter: ((NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid)
                                            Buffers: shared hit=12
                                            ->  Nested Loop  (cost=0.72..16.79 rows=1 width=1235) (actual time=0.221..0.317 rows=1 loops=1)
                                                  Buffers: shared hit=12
                                                  ->  Index Scan using record_shares_principal on record_shares s  (cost=0.29..8.31 rows=1 width=32) (actual time=0.063..0.079 rows=2 loops=1)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (principal_id = 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid))
                                                        Filter: (principal_kind = 'user'::text)
                                                        Buffers: shared hit=4
                                                  ->  Index Scan using records_pkey on records r_2  (cost=0.42..8.46 rows=1 width=1235) (actual time=0.115..0.115 rows=0 loops=2)
                                                        Index Cond: ((tenant_id = '809d4580-aaed-4156-5abc-38d58f77f840'::uuid) AND (id = s.record_id))
                                                        Filter: ((requester_id IS DISTINCT FROM 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) AND (assignee_id IS DISTINCT FROM 'f222cde6-739e-cb4e-eaa6-14c1a2bed245'::uuid) AND (entity_definition_id = '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2'::uuid) AND (ROW(updated_at, id) < ROW('2026-02-15 00:27:00.844898+00'::timestamp with time zone, '09b90e91-b333-8cdc-5df8-c0dac0aaa909'::uuid)))
                                                        Rows Removed by Filter: 0
                                                        Buffers: shared hit=8
        ->  Index Scan using entity_definitions_pkey on entity_definitions d  (cost=0.28..8.30 rows=1 width=39) (actual time=0.002..0.002 rows=1 loops=51)
              Index Cond: (tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)
              Buffers: shared hit=153
  ->  Index Scan using groups_live_key on groups g  (cost=0.30..8.32 rows=1 width=40) (actual time=0.006..0.006 rows=1 loops=51)
        Index Cond: ((tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid) AND (id = x.assignment_group_id))
        Buffers: shared hit=153
Settings: effective_cache_size = '6GB', jit = 'off'
Planning:
  Buffers: shared hit=593
Planning Time: 4.057 ms
Execution Time: 2.832 ms
```
