-- The per-tenant INSERT's dedupe probe, for the biggest tenant, as
-- nava_migrator under the backfill's canonical policies (rolled back).
\set ON_ERROR_STOP on
SET ROLE nava_migrator;
BEGIN;
CREATE POLICY p_t ON tenants FOR SELECT TO nava_migrator USING (true);
CREATE POLICY p_v ON entity_definition_versions FOR SELECT TO nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY p_s ON audit_log FOR SELECT TO nava_migrator
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
SELECT set_config('app.tenant_id', (SELECT id::text FROM tenants ORDER BY id LIMIT 1), true);
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT v.id FROM entity_definition_versions v
WHERE v.tenant_id = current_setting('app.tenant_id')::uuid
  AND NOT EXISTS (SELECT 1 FROM audit_log l WHERE l.tenant_id = v.tenant_id
      AND l.subject_type = 'entity_definition' AND l.subject_id = v.definition_id AND l.version_after = v.version);
ROLLBACK;
