-- Run as nava_app under forced RLS, custom plans. :label names the phase.
SET ROLE nava_app;
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true) \gset
SET LOCAL plan_cache_mode = force_custom_plan;
\echo == J0 JSONB closed: data @> {priority:3}, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT id FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND r.data @> '{"priority":3}' ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == J1 JSONB window: OR NOT data ? priority, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT id FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND (r.data @> '{"priority":3}' OR NOT (r.data ? 'priority')) ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == J0c / J1c capped count 1001
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT count(*) FROM (SELECT 1 FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND r.data @> '{"priority":3}' LIMIT 1001) s;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT count(*) FROM (SELECT 1 FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND (r.data @> '{"priority":3}' OR NOT (r.data ? 'priority')) LIMIT 1001) s;
\echo == S0 side table closed: EXISTS value = 3, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT id FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id
               AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
 ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == S1 side table window: EXISTS OR NOT data ? priority, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT id FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id
               AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR NOT (r.data ? 'priority'))
 ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == S0c / S1c capped count
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT count(*) FROM (SELECT 1 FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id
               AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) LIMIT 1001) s;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT count(*) FROM (SELECT 1 FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id
               AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR NOT (r.data ? 'priority')) LIMIT 1001) s;
\echo == S2 window, absent set bounded by schema_backfill_failures (proposed)
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF, SUMMARY ON) SELECT id FROM records r
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001'
   AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id
               AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
        OR (EXISTS (SELECT 1 FROM schema_backfill_failures x WHERE x.tenant_id = r.tenant_id
               AND x.entity_definition_id = r.entity_definition_id AND x.target_version = 3 AND x.record_id = r.id)
            AND NOT (r.data ? 'priority')))
 ORDER BY updated_at DESC, id DESC LIMIT 25;
COMMIT;
