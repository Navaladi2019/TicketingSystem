-- Round 2: the D12 switch point. :k records lack priority; all are pending;
-- the default 3 is stored nowhere (the sparse worst case).
CREATE TABLE IF NOT EXISTS schema_backfill_pending (tenant_id uuid, entity_definition_id uuid, target_version int, record_id uuid,
  PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id));
ALTER TABLE schema_backfill_pending ENABLE ROW LEVEL SECURITY; ALTER TABLE schema_backfill_pending FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS p ON schema_backfill_pending;
CREATE POLICY p ON schema_backfill_pending TO nava_app, nava_worker USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT, DELETE ON schema_backfill_pending TO nava_app, nava_worker;
ALTER TABLE records DISABLE TRIGGER rfv_maint;
UPDATE records SET data = data || jsonb_build_object('priority', (ARRAY[1,2,4,5])[1 + (substr(number,3)::int % 4)])
 WHERE tenant_id = u(1,1) AND NOT data ? 'priority';
UPDATE records SET data = data - 'priority' WHERE tenant_id = u(1,1) AND substr(number,3)::int % (100000 / :k) = 0;
ALTER TABLE records ENABLE TRIGGER rfv_maint;
TRUNCATE rfv_num, schema_backfill_pending, schema_backfill_failures;
INSERT INTO rfv_num SELECT tenant_id, u(1,99), id, 0, (data->>'priority')::float8 FROM records WHERE tenant_id = u(1,1) AND data ? 'priority';
INSERT INTO schema_backfill_pending SELECT tenant_id, entity_definition_id, 3, id FROM records WHERE tenant_id = u(1,1) AND NOT data ? 'priority';
ANALYZE records; ANALYZE rfv_num; ANALYZE schema_backfill_pending; ANALYZE schema_backfill_failures;
SELECT count(*) AS pending FROM schema_backfill_pending;
SET ROLE nava_app;
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true) \g /dev/null
SET LOCAL plan_cache_mode = force_custom_plan;
\echo == OR page
EXPLAIN (ANALYZE, COSTS OFF) SELECT id FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR NOT (r.data ? 'priority')) ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == OR count
EXPLAIN (ANALYZE, COSTS OFF) SELECT count(*) FROM (SELECT 1 FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR NOT (r.data ? 'priority')) LIMIT 1001) s;
\echo == UNION page
EXPLAIN (ANALYZE, COSTS OFF) SELECT id FROM (SELECT r.id, r.updated_at FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
  UNION ALL
  SELECT r.id, r.updated_at FROM (SELECT record_id FROM schema_backfill_pending WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND target_version = 3
                                  UNION ALL SELECT record_id FROM schema_backfill_failures WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND target_version = 3) x
   JOIN records r ON r.tenant_id = '00000001-0000-7000-8000-000000000001' AND r.id = x.record_id
   WHERE r.entity_definition_id = '00000002-0000-7000-8000-000000000001' AND NOT (r.data ? 'priority') AND NOT EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)) u ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == UNION count
EXPLAIN (ANALYZE, COSTS OFF) SELECT count(*) FROM (SELECT 1 FROM (SELECT r.id, r.updated_at FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
  UNION ALL
  SELECT r.id, r.updated_at FROM (SELECT record_id FROM schema_backfill_pending WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND target_version = 3
                                  UNION ALL SELECT record_id FROM schema_backfill_failures WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND target_version = 3) x
   JOIN records r ON r.tenant_id = '00000001-0000-7000-8000-000000000001' AND r.id = x.record_id
   WHERE r.entity_definition_id = '00000002-0000-7000-8000-000000000001' AND NOT (r.data ? 'priority') AND NOT EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)) u LIMIT 1001) s;
COMMIT;
