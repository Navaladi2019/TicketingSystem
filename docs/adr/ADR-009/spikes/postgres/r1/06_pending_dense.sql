-- Dense case for a pending-set-driven branch: 60,000 pending ids (scenario A size).
CREATE TABLE schema_backfill_pending (tenant_id uuid, entity_definition_id uuid, target_version int, record_id uuid,
  PRIMARY KEY (tenant_id, entity_definition_id, target_version, record_id));
ALTER TABLE schema_backfill_pending ENABLE ROW LEVEL SECURITY; ALTER TABLE schema_backfill_pending FORCE ROW LEVEL SECURITY;
CREATE POLICY p ON schema_backfill_pending TO nava_app, nava_worker USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT, DELETE ON schema_backfill_pending TO nava_app, nava_worker;
\timing on
INSERT INTO schema_backfill_pending SELECT tenant_id, entity_definition_id, 3, id FROM records WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND substr(number,3)::int % 5 >= 2;
\timing off
ANALYZE schema_backfill_pending;
SELECT pg_size_pretty(pg_total_relation_size('schema_backfill_pending')) pending_size;
SET ROLE nava_app;
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true);
SET LOCAL plan_cache_mode = force_custom_plan;
\echo == P1 UNION ALL, pending branch, 60k pending, page (no NOT ? filter: worst-case join+sort)
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM (
  SELECT r.id, r.updated_at FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3)
  UNION ALL
  SELECT r.id, r.updated_at FROM schema_backfill_pending x JOIN records r ON r.tenant_id = x.tenant_id AND r.id = x.record_id
   WHERE x.tenant_id = '00000001-0000-7000-8000-000000000001' AND x.entity_definition_id = '00000002-0000-7000-8000-000000000001' AND x.target_version = 3 AND r.tenant_id = '00000001-0000-7000-8000-000000000001' AND r.entity_definition_id = '00000002-0000-7000-8000-000000000001'
) u ORDER BY updated_at DESC, id DESC LIMIT 25;
\echo == P2 OR-EXISTS(pending) on the ordered walk, 60k pending, page
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM records r WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND entity_definition_id = '00000002-0000-7000-8000-000000000001' AND (EXISTS (SELECT 1 FROM rfv_num v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = '00000063-0000-7000-8000-000000000001' AND v.value = 3) OR EXISTS (
  SELECT 1 FROM schema_backfill_pending x WHERE x.tenant_id = r.tenant_id AND x.entity_definition_id = r.entity_definition_id AND x.target_version = 3 AND x.record_id = r.id))
 ORDER BY updated_at DESC, id DESC LIMIT 25;
COMMIT;
