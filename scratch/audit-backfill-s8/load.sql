-- :tenants tenants, :types types each, :versions versions per type; v1 has
-- no publisher in every other tenant (onboarding), the rest name one.
\set ON_ERROR_STOP on
TRUNCATE audit_log, entity_definition_versions, entity_definitions, principals, tenants CASCADE;
INSERT INTO tenants SELECT gen_random_uuid() FROM generate_series(1, :tenants);
INSERT INTO principals SELECT id, id FROM tenants;
INSERT INTO entity_definitions SELECT t.id, gen_random_uuid() FROM tenants t, generate_series(1, :types);
INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256, published_by, published_at)
SELECT d.tenant_id, gen_random_uuid(), d.id, v, '{"type":"object"}', sha256('{"type":"object"}'),
       CASE WHEN v = 1 AND hashtext(d.tenant_id::text) % 2 = 0 THEN NULL ELSE d.tenant_id END,
       now() - make_interval(days => 100 - v)
FROM entity_definitions d, generate_series(1, :versions) v;
ANALYZE;
SELECT count(*) AS tenants FROM tenants;
SELECT count(*) AS versions FROM entity_definition_versions;
