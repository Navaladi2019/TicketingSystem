# Solution Architect plan-shape prototype: steered slot order key (TICK-1206, 2026-11-28)

Plan shape only, not a scored latency run: no host gate, toy seed. PG 16.10 (postgres:16.10), shared_buffers 512 MB, work_mem 4 MB, jit off. 800k rows: one 400k type (50 groups; newest 100k rows carry fx=4, older rows fx=3 at 50% of the type, oldest 8k fx=40), plus 400k noise rows. Forced RLS on tenant_id via current_setting, queries as a non-owner role. Scripts: setup.sql, steer_idx.sql, leak.sql, shuffle.sql (the random-order rewrite failed on container /dev/shm; only the 200k-row non-HOT update churn ran, updated_at correlation fell to 0.50). Container removed after the run.

## setup.sql
```sql
CREATE ROLE app LOGIN;
CREATE TABLE records(
  id bigint PRIMARY KEY, tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL,
  assignment_group_id uuid, updated_at timestamptz NOT NULL, fx_num_2 double precision, pad text);
-- tenant A, type T: 400k rows, 50 groups; newest 100k carry fx=4 (recency anti-correlation),
-- older 300k: 2/3 carry fx=3 (=200k, 50% of type), 8k oldest carry fx=40, rest spread.
INSERT INTO records
SELECT g, '00000000-0000-0000-0000-00000000000a', '00000000-0000-0000-0000-0000000000d1',
  ('00000000-0000-0000-0000-' || lpad((g % 50)::text, 12, '0'))::uuid,
  timestamptz '2026-01-01' + g * interval '10 seconds',
  CASE WHEN g > 300000 THEN 4 WHEN g <= 8000 THEN 40 WHEN g % 3 <> 0 THEN 3 ELSE (g % 97) + 100 END,
  repeat('x', 200)
FROM generate_series(1, 400000) g;
-- noise: same tenant other type 200k, other tenant 200k
INSERT INTO records
SELECT 400000 + g, CASE WHEN g % 2 = 0 THEN '00000000-0000-0000-0000-00000000000a'::uuid ELSE '00000000-0000-0000-0000-00000000000b'::uuid END,
  '00000000-0000-0000-0000-0000000000d2',
  ('00000000-0000-0000-0000-' || lpad((g % 50)::text, 12, '0'))::uuid,
  timestamptz '2026-01-01' + g * interval '10 seconds', (g % 10), repeat('x', 200)
FROM generate_series(1, 400000) g;
CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);
CREATE INDEX records_list_group   ON records (tenant_id, entity_definition_id, assignment_group_id, updated_at DESC, id DESC);
-- A3-2 rev 0 slot index (as measured on TICK-1205)
CREATE INDEX slot_grp ON records (tenant_id, entity_definition_id, fx_num_2, assignment_group_id, updated_at) WHERE fx_num_2 IS NOT NULL;
CREATE STATISTICS st_slot (mcv) ON tenant_id, entity_definition_id, fx_num_2 FROM records;
ALTER TABLE records ENABLE ROW LEVEL SECURITY; ALTER TABLE records FORCE ROW LEVEL SECURITY;
CREATE POLICY p ON records USING (tenant_id = current_setting('app.tenant')::uuid);
GRANT SELECT ON records TO app;
VACUUM ANALYZE records;
```
## steer_idx.sql
```sql
CREATE INDEX slot_grp_steer ON records (tenant_id, entity_definition_id, fx_num_2, assignment_group_id, (updated_at AT TIME ZONE 'UTC') DESC, id DESC) WHERE fx_num_2 IS NOT NULL;
CREATE INDEX slot_plain_steer ON records (tenant_id, entity_definition_id, fx_num_2, (updated_at AT TIME ZONE 'UTC') DESC, id DESC) WHERE fx_num_2 IS NOT NULL;
ANALYZE records;
SELECT p.proname, p.provolatile, p.proleakproof FROM pg_proc p WHERE p.oid = 'timezone(text,timestamptz)'::regprocedure;
```
## leak.sql
```sql
CREATE FUNCTION nava_list_key(timestamptz) RETURNS timestamp
  LANGUAGE plpgsql IMMUTABLE STRICT LEAKPROOF PARALLEL SAFE AS $$ BEGIN RETURN $1 AT TIME ZONE 'UTC'; END $$;
CREATE INDEX slot_plain_key ON records (tenant_id, entity_definition_id, fx_num_2, nava_list_key(updated_at) DESC, id DESC) WHERE fx_num_2 IS NOT NULL;
CREATE INDEX slot_grp_key ON records (tenant_id, entity_definition_id, fx_num_2, assignment_group_id, nava_list_key(updated_at) DESC, id DESC) WHERE fx_num_2 IS NOT NULL;
DROP INDEX slot_grp_steer, slot_plain_steer;
ANALYZE records;
SET ROLE app; SET app.tenant = '00000000-0000-0000-0000-00000000000a';
\echo === LEAKPROOF key, no ACL, page 2 cursor
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF)
SELECT id FROM records r WHERE r.entity_definition_id='00000000-0000-0000-0000-0000000000d1' AND r.fx_num_2 = 3
  AND (nava_list_key(updated_at), id) < (timestamp '2026-01-31 00:00', 260000) ORDER BY nava_list_key(updated_at) DESC, id DESC LIMIT 51;
\echo === LEAKPROOF key, merge arm with cursor, group 2
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, TIMING OFF)
SELECT id FROM records r WHERE r.entity_definition_id='00000000-0000-0000-0000-0000000000d1' AND r.fx_num_2 = 3 AND r.assignment_group_id='00000000-0000-0000-0000-000000000002'
  AND (nava_list_key(updated_at), id) < (timestamp '2026-01-31 00:00', 260000) ORDER BY nava_list_key(updated_at) DESC, id DESC LIMIT 51;
```
## shuffle.sql
```sql
CREATE TABLE tmp AS SELECT * FROM records ORDER BY random();
TRUNCATE records; INSERT INTO records SELECT * FROM tmp; DROP TABLE tmp;
-- simulate update churn: 300k non-HOT updates to fx=4 rows' pad, no VACUUM FULL
UPDATE records SET pad = repeat('y',200) WHERE id > 200000 AND id <= 400000;
VACUUM ANALYZE records;
SELECT attname, correlation FROM pg_stats WHERE tablename='records' AND attname IN ('updated_at','id');
```
## Output: steered indexes on the physically ordered heap (q_steer.sql)
Not captured verbatim. Summary: no-ACL broad page on (updated_at AT TIME ZONE 'UTC') key, Index Scan slot_plain_steer, 51 rows, 0.074 ms; same with row cursor, Filter not Index Cond (timezone() is not leakproof), 27,200 rows removed, 7.2 ms; 25-arm merge, 25 Index Scans on slot_grp_steer, 3 rows each, 0.775 ms.

## Output: leakproof order key, cursor (leak_out.txt)
```
## leakproof cursor
                                                                                                                                                 QUERY PLAN                                                                                                                                                  
-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=124 dirtied=2
   ->  Index Scan using slot_plain_key on records r (actual rows=51 loops=1)
         Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (ROW(nava_list_key(updated_at), id) < ROW('2026-01-31 00:00:00'::timestamp without time zone, 260000)))
         Buffers: shared hit=124 dirtied=2
 Planning:
   Buffers: shared hit=317
 Planning Time: 1.030 ms
 Execution Time: 1.094 ms
(9 rows)

```
## Output: after churn (shuf_out.txt)
```
=== SHUFFLED baseline merge a50 (ORDER BY updated_at)
                                                                                                                                         QUERY PLAN                                                                                                                                          
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=1776 dirtied=3
   ->  Merge Append (actual rows=51 loops=1)
         Sort Key: r.updated_at DESC, r.id DESC
         Buffers: shared hit=1776 dirtied=3
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r.updated_at DESC, r.id DESC
                     Presorted Key: r.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000000'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_1.updated_at DESC, r_1.id DESC
                     Presorted Key: r_1.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_1 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000002'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_2.updated_at DESC, r_2.id DESC
                     Presorted Key: r_2.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_2 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000004'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_3.updated_at DESC, r_3.id DESC
                     Presorted Key: r_3.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_3 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000006'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_4.updated_at DESC, r_4.id DESC
                     Presorted Key: r_4.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_4 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000008'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72 dirtied=1
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_5.updated_at DESC, r_5.id DESC
                     Presorted Key: r_5.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72 dirtied=1
                     ->  Index Scan Backward using slot_grp on records r_5 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000010'::uuid))
                           Buffers: shared hit=72 dirtied=1
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_6.updated_at DESC, r_6.id DESC
                     Presorted Key: r_6.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_6 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000012'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_7.updated_at DESC, r_7.id DESC
                     Presorted Key: r_7.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_7 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000014'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72 dirtied=1
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_8.updated_at DESC, r_8.id DESC
                     Presorted Key: r_8.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72 dirtied=1
                     ->  Index Scan Backward using slot_grp on records r_8 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000016'::uuid))
                           Buffers: shared hit=72 dirtied=1
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_9.updated_at DESC, r_9.id DESC
                     Presorted Key: r_9.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_9 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000018'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_10.updated_at DESC, r_10.id DESC
                     Presorted Key: r_10.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_10 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000020'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_11.updated_at DESC, r_11.id DESC
                     Presorted Key: r_11.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_11 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000022'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_12.updated_at DESC, r_12.id DESC
                     Presorted Key: r_12.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_12 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000024'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_13.updated_at DESC, r_13.id DESC
                     Presorted Key: r_13.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_13 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000026'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_14.updated_at DESC, r_14.id DESC
                     Presorted Key: r_14.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_14 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000028'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_15.updated_at DESC, r_15.id DESC
                     Presorted Key: r_15.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_15 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000030'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_16.updated_at DESC, r_16.id DESC
                     Presorted Key: r_16.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_16 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000032'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72 dirtied=1
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_17.updated_at DESC, r_17.id DESC
                     Presorted Key: r_17.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72 dirtied=1
                     ->  Index Scan Backward using slot_grp on records r_17 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000034'::uuid))
                           Buffers: shared hit=72 dirtied=1
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_18.updated_at DESC, r_18.id DESC
                     Presorted Key: r_18.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_18 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000036'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_19.updated_at DESC, r_19.id DESC
                     Presorted Key: r_19.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_19 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000038'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_20.updated_at DESC, r_20.id DESC
                     Presorted Key: r_20.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_20 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000040'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_21.updated_at DESC, r_21.id DESC
                     Presorted Key: r_21.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_21 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000042'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_22.updated_at DESC, r_22.id DESC
                     Presorted Key: r_22.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_22 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000044'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_23.updated_at DESC, r_23.id DESC
                     Presorted Key: r_23.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r_23 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000046'::uuid))
                           Buffers: shared hit=72
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=69
               ->  Incremental Sort (actual rows=3 loops=1)
                     Sort Key: r_24.updated_at DESC, r_24.id DESC
                     Presorted Key: r_24.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=69
                     ->  Index Scan Backward using slot_grp on records r_24 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000048'::uuid))
                           Buffers: shared hit=69
 Planning:
   Buffers: shared hit=294 read=28
 Planning Time: 21.436 ms
 Execution Time: 11.887 ms
(259 rows)

=== SHUFFLED steered merge a50 (ORDER BY nava_list_key)
                                                                                                                                      QUERY PLAN                                                                                                                                       
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=264 read=5
   ->  Merge Append (actual rows=51 loops=1)
         Sort Key: (nava_list_key(r.updated_at)) DESC, r.id DESC
         Buffers: shared hit=264 read=5
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=24 read=5
               ->  Index Scan using slot_grp_key on records r (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000000'::uuid))
                     Buffers: shared hit=24 read=5
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_1 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000002'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_2 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000004'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_3 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000006'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_4 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000008'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_5 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000010'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_6 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000012'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_7 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000014'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_8 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000016'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_9 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000018'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_10 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000020'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_11 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000022'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_12 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000024'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_13 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000026'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_14 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000028'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_15 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000030'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_16 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000032'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_17 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000034'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_18 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000036'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_19 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000038'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_20 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000040'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_21 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000042'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_22 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000044'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_23 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000046'::uuid))
                     Buffers: shared hit=10
         ->  Limit (actual rows=3 loops=1)
               Buffers: shared hit=10
               ->  Index Scan using slot_grp_key on records r_24 (actual rows=3 loops=1)
                     Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000048'::uuid))
                     Buffers: shared hit=10
 Planning:
   Buffers: shared hit=4
 Planning Time: 3.375 ms
 Execution Time: 1.334 ms
(134 rows)

=== SHUFFLED baseline merge a6 (3 groups)
                                                                                                                                         QUERY PLAN                                                                                                                                          
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=113
   ->  Merge Append (actual rows=51 loops=1)
         Sort Key: r.updated_at DESC, r.id DESC
         Buffers: shared hit=113
         ->  Limit (actual rows=18 loops=1)
               Buffers: shared hit=37
               ->  Incremental Sort (actual rows=18 loops=1)
                     Sort Key: r.updated_at DESC, r.id DESC
                     Presorted Key: r.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=37
                     ->  Index Scan Backward using slot_grp on records r (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000000'::uuid))
                           Buffers: shared hit=37
         ->  Limit (actual rows=17 loops=1)
               Buffers: shared hit=39
               ->  Incremental Sort (actual rows=17 loops=1)
                     Sort Key: r_1.updated_at DESC, r_1.id DESC
                     Presorted Key: r_1.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=39
                     ->  Index Scan Backward using slot_grp on records r_1 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000002'::uuid))
                           Buffers: shared hit=39
         ->  Limit (actual rows=18 loops=1)
               Buffers: shared hit=37
               ->  Incremental Sort (actual rows=18 loops=1)
                     Sort Key: r_2.updated_at DESC, r_2.id DESC
                     Presorted Key: r_2.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=37
                     ->  Index Scan Backward using slot_grp on records r_2 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000004'::uuid))
                           Buffers: shared hit=37
 Planning Time: 0.523 ms
 Execution Time: 0.204 ms
(37 rows)

=== SHUFFLED steered no ACL
                                                                                            QUERY PLAN                                                                                            
--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=107 dirtied=1
   ->  Index Scan using slot_plain_key on records r (actual rows=51 loops=1)
         Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision))
         Buffers: shared hit=107 dirtied=1
 Planning Time: 0.191 ms
 Execution Time: 0.290 ms
(7 rows)

=== SHUFFLED baseline no ACL
                                                                        QUERY PLAN                                                                         
-----------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=202773 dirtied=154
   ->  Index Scan using records_list_updated on records r (actual rows=51 loops=1)
         Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid))
         Filter: (fx_num_2 = '3'::double precision)
         Rows Removed by Filter: 100026
         Buffers: shared hit=202773 dirtied=154
 Planning Time: 0.197 ms
 Execution Time: 191.369 ms
(9 rows)

```
## Output: baseline after churn (base_out.txt)
```
## base (physically ordered heap)
=== BASELINE merge, 3 groups, broad value 3, ORDER BY updated_at
                                                                                                                                         QUERY PLAN                                                                                                                                          
---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=148
   ->  Merge Append (actual rows=51 loops=1)
         Sort Key: r.updated_at DESC, r.id DESC
         Buffers: shared hit=148
         ->  Limit (actual rows=18 loops=1)
               Buffers: shared hit=72
               ->  Incremental Sort (actual rows=18 loops=1)
                     Sort Key: r.updated_at DESC, r.id DESC
                     Presorted Key: r.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=72
                     ->  Index Scan Backward using slot_grp on records r (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000001'::uuid))
                           Buffers: shared hit=69
         ->  Limit (actual rows=17 loops=1)
               Buffers: shared hit=39
               ->  Incremental Sort (actual rows=17 loops=1)
                     Sort Key: r_1.updated_at DESC, r_1.id DESC
                     Presorted Key: r_1.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=39
                     ->  Index Scan Backward using slot_grp on records r_1 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000002'::uuid))
                           Buffers: shared hit=39
         ->  Limit (actual rows=18 loops=1)
               Buffers: shared hit=37
               ->  Incremental Sort (actual rows=18 loops=1)
                     Sort Key: r_2.updated_at DESC, r_2.id DESC
                     Presorted Key: r_2.updated_at
                     Full-sort Groups: 1  Sort Method: quicksort  Average Memory: 26kB  Peak Memory: 26kB
                     Buffers: shared hit=37
                     ->  Index Scan Backward using slot_grp on records r_2 (actual rows=33 loops=1)
                           Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid) AND (fx_num_2 = '3'::double precision) AND (assignment_group_id = '00000000-0000-0000-0000-000000000004'::uuid))
                           Buffers: shared hit=37
 Planning:
   Buffers: shared hit=322
 Planning Time: 1.336 ms
 Execution Time: 0.492 ms
(39 rows)

=== BASELINE no ACL broad value 3
                                                                        QUERY PLAN                                                                         
-----------------------------------------------------------------------------------------------------------------------------------------------------------
 Limit (actual rows=51 loops=1)
   Buffers: shared hit=153706
   ->  Index Scan using records_list_updated on records r (actual rows=51 loops=1)
         Index Cond: ((tenant_id = (current_setting('app.tenant'::text))::uuid) AND (entity_definition_id = '00000000-0000-0000-0000-0000000000d1'::uuid))
         Filter: (fx_num_2 = '3'::double precision)
         Rows Removed by Filter: 100026
         Buffers: shared hit=153706
 Planning Time: 0.249 ms
 Execution Time: 77.465 ms
(9 rows)

```
