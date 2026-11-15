SET ROLE nava_app;
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true);
SET LOCAL plan_cache_mode = force_custom_plan;
\echo == B-S0 closed window, side table, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == B-S1 RFC D12 shape: side OR NOT data ? f, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR NOT (r.data ? 'priority')) ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == B-S1c RFC D12 shape, capped count
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT count(*) FROM (SELECT 1 FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR NOT (r.data ? 'priority')) LIMIT 1001) s;
\echo == B-J1 JSONB path D12 shape, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (r.data @> '{"priority":3}' OR NOT (r.data ? 'priority')) ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == B-S3 UNION ALL, absent branch driven by schema_backfill_failures PK, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM (
  SELECT r.id, r.updated_at FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
  UNION ALL
  SELECT r.id, r.updated_at FROM schema_backfill_failures x JOIN records r ON r.tenant_id = x.tenant_id AND r.id = x.record_id
   WHERE x.tenant_id = '00000001-0000-7000-8000-000000000001' AND x.entity_definition_id = '00000002-0000-7000-8000-000000000001' AND x.target_version = 3 AND r.tenant_id = '00000001-0000-7000-8000-000000000001' AND r.entity_definition_id = '00000002-0000-7000-8000-000000000001' AND NOT (r.data ? 'priority')
) u ORDER BY updated_at DESC, id DESC LIMIT 25;
COMMIT;
RESET ROLE;
CREATE INDEX records_def_schema_version ON records (tenant_id, entity_definition_id, schema_version);
ANALYZE records;
SET ROLE nava_app;
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true);
SET LOCAL plan_cache_mode = force_custom_plan;
\echo == B-S4 UNION ALL, absent branch driven by (tenant, def, schema_version) index, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM (
  SELECT r.id, r.updated_at FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
  UNION ALL
  SELECT r.id, r.updated_at FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND schema_version < 3 AND NOT (r.data ? 'priority')
) u ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == B-S5 single OR with schema_version bound (no UNION), page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR (schema_version < 3 AND NOT (r.data ? 'priority'))) ORDER BY updated_at DESC, id DESC LIMIT 25;
COMMIT;
RESET ROLE;
SELECT pg_size_pretty(pg_relation_size('records_def_schema_version')) sv_idx, pg_size_pretty(pg_relation_size('records_list_updated')) list_idx;
DROP INDEX records_def_schema_version;
