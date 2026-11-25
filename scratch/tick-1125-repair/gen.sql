-- TICK-1125 bench data: one tenant, a type with 2 refused versions and :n
-- records of about 300 bytes with one create entry each, and 100,000 audit
-- entries of another type of the same tenant. Run by bench.py on a database
-- migrated through 20261120160200.
SET session_replication_role = replica;
INSERT INTO tenants (id, state) VALUES (:'tenant', 'active');
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
VALUES (:'tenant', :'type', 'ticket', 'Ticket', 'TCK', 2), (:'tenant', :'other', 'other', 'Other', 'OTH', 1);
INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
SELECT :'tenant', gen_random_uuid(), :'type', v, s::json, sha256(convert_to(s, 'UTF8')), '{"state": "state"}',
    '{"initial": "new", "states": ["new"], "transitions": []}', CASE v WHEN 1 THEN 'initial' ELSE 'compatible' END, :'tenant'
FROM generate_series(1, 2) v,
    (VALUES ('{"type":"object","additionalProperties":false,"properties":{"state":{"type":"string"},"org_unit_id":{"type":"string"},"summary":{"type":"string"}}}')) x (s);
INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, data, version)
SELECT :'tenant', gen_random_uuid(), :'type', 2, 'TCK-' || g, 'new',
    jsonb_build_object('org_unit_id', 'unit ' || g, 'summary', repeat('x', 250)), 1
FROM generate_series(1, :n) g;
INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id, version_after,
    actor_system, source, diff, diff_salt, diff_sha256)
SELECT tenant_id, gen_random_uuid(), now() - interval '1 day', 'record', 'create', 'record', id, entity_definition_id, 1, 'system', 'system',
    d::json, s, sha256(s || convert_to(d, 'UTF8'))
FROM (SELECT r.*, '[{"op":"add","path":"","value":' || data::text || '}]' AS d, substring(sha256(uuid_send(id)) FROM 1 FOR 16) AS s
      FROM records r WHERE entity_definition_id = :'type') x;
INSERT INTO audit_log (tenant_id, id, occurred_at, domain, action, subject_type, subject_id, entity_definition_id, version_after,
    actor_system, source, diff, diff_salt, diff_sha256)
SELECT :'tenant', gen_random_uuid(), now() - interval '1 day', 'record', 'create', 'record', gen_random_uuid(), :'other', 1, 'system', 'system',
    d::json, s, sha256(s || convert_to(d, 'UTF8'))
FROM (SELECT '[{"op":"add","path":"","value":{"summary":"' || repeat('x', 250) || g || '"}}]' AS d,
        '\x00112233445566778899aabbccddeeff'::bytea AS s
      FROM generate_series(1, 100000) g) x;
SET session_replication_role = DEFAULT;
VACUUM ANALYZE records, audit_log, entity_definition_versions;
SELECT round(avg(octet_length(data::text))) || ' bytes of data' FROM records WHERE entity_definition_id = :'type';
