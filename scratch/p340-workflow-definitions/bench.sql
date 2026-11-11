-- P3-40 (TICK-709) evidence: the sink's two per-write lookups on the
-- P3-40 tables, before and after their index, at a realistic shape.
-- Run: sh scratch/p340-workflow-definitions/run.sh (throwaway postgres:16.10).
-- Shape: 2,000 tenants x 5 entity types; 10 workflow definitions per tenant
-- (20,000), 3 published versions each (60,000 version rows), current = 3;
-- 2,000,000 workflow_ref rows (1,000 per tenant over 200 records), 5% open,
-- of which half wait on record_changed (50,000 waiting).
\timing off
SET max_parallel_workers_per_gather = 0;

CREATE TABLE workflow_definitions (
    tenant_id uuid NOT NULL, id uuid NOT NULL, key text NOT NULL, entity_definition_id uuid NOT NULL,
    current_version integer NOT NULL, created_by uuid, created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id), UNIQUE (tenant_id, key));
CREATE TABLE workflow_definition_versions (
    tenant_id uuid NOT NULL, id uuid NOT NULL, definition_id uuid NOT NULL, version integer NOT NULL,
    body json NOT NULL, body_sha256 bytea NOT NULL, step_ids jsonb NOT NULL, trigger_on text NOT NULL,
    trigger_entity_definition_id uuid NOT NULL, published_by uuid, published_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, definition_id, version), UNIQUE (tenant_id, id));
CREATE TABLE workflow_ref (
    tenant_id uuid NOT NULL, outbox_id uuid NOT NULL, record_id uuid NOT NULL, kind text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), closed_at timestamptz, record_changed_signal_id uuid,
    PRIMARY KEY (tenant_id, outbox_id));
CREATE INDEX workflow_ref_record ON workflow_ref (tenant_id, record_id, kind, created_at DESC);

CREATE TEMP TABLE t AS SELECT gen_random_uuid() AS tenant_id, n FROM generate_series(1, 2000) n;
CREATE TEMP TABLE e AS SELECT t.tenant_id, gen_random_uuid() AS id, k FROM t, generate_series(1, 5) k;

INSERT INTO workflow_definitions (tenant_id, id, key, entity_definition_id, current_version)
SELECT t.tenant_id, gen_random_uuid(), 'wf_' || d, (SELECT id FROM e WHERE e.tenant_id = t.tenant_id AND e.k = 1 + d % 5), 3
FROM t, generate_series(1, 10) d;

INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256, step_ids,
                                          trigger_on, trigger_entity_definition_id)
SELECT d.tenant_id, gen_random_uuid(), d.id, v, b.body::json, sha256(convert_to(b.body, 'UTF8')),
       '{"start": "00000000-0000-4000-8000-000000000001"}',
       CASE WHEN v % 2 = 0 THEN 'record.updated' ELSE 'record.created' END, d.entity_definition_id
FROM workflow_definitions d, generate_series(1, 3) v,
     LATERAL (SELECT '{"format": "v1", "start": "s", "steps": [' || repeat('{"key": "s", "type": "end"},', 20) || '{"key": "e", "type": "end"}]}' AS body) b;

INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, closed_at, record_changed_signal_id)
SELECT t.tenant_id, gen_random_uuid(), r.record_id, 'nava.interpreter',
       CASE WHEN i % 20 = 0 THEN NULL ELSE now() END,
       CASE WHEN i % 40 = 0 THEN gen_random_uuid() END
FROM t, generate_series(1, 1000) i,
     LATERAL (SELECT ('00000000-0000-4000-8000-' || lpad((i % 200)::text, 12, '0'))::uuid AS record_id) r;

VACUUM ANALYZE workflow_definitions, workflow_definition_versions, workflow_ref;

SELECT 'rows' AS what, (SELECT count(*) FROM workflow_definitions) AS defs,
       (SELECT count(*) FROM workflow_definition_versions) AS versions, (SELECT count(*) FROM workflow_ref) AS refs,
       (SELECT count(*) FROM workflow_ref WHERE closed_at IS NULL AND record_changed_signal_id IS NOT NULL) AS waiting;

-- The probe tenant, entity type and record.
SELECT tenant_id AS probe_tenant FROM t WHERE n = 1000 \gset
SELECT entity_definition_id AS probe_entity FROM workflow_definitions WHERE tenant_id = :'probe_tenant' AND key = 'wf_1' \gset
SELECT record_id AS probe_record FROM workflow_ref WHERE tenant_id = :'probe_tenant' AND closed_at IS NULL AND record_changed_signal_id IS NOT NULL LIMIT 1 \gset

\echo '== (1) trigger lookup, BEFORE workflow_definition_versions_trigger'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT v.id, v.definition_id FROM workflow_definition_versions v
JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id AND d.current_version = v.version
WHERE v.tenant_id = :'probe_tenant' AND v.trigger_entity_definition_id = :'probe_entity' AND v.trigger_on = 'record.created';

\echo '== (2) waiting refs on a record, BEFORE workflow_ref_waiting'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT outbox_id, record_changed_signal_id FROM workflow_ref
WHERE tenant_id = :'probe_tenant' AND record_id = :'probe_record' AND closed_at IS NULL AND record_changed_signal_id IS NOT NULL;

CREATE INDEX workflow_definition_versions_trigger ON workflow_definition_versions
    (tenant_id, trigger_entity_definition_id, trigger_on) INCLUDE (definition_id, version, id);
CREATE INDEX workflow_ref_waiting ON workflow_ref (tenant_id, record_id)
    WHERE closed_at IS NULL AND record_changed_signal_id IS NOT NULL;
VACUUM ANALYZE workflow_definition_versions, workflow_ref;

\echo '== (1) trigger lookup, AFTER'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT v.id, v.definition_id FROM workflow_definition_versions v
JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id AND d.current_version = v.version
WHERE v.tenant_id = :'probe_tenant' AND v.trigger_entity_definition_id = :'probe_entity' AND v.trigger_on = 'record.created';

\echo '== (2) waiting refs on a record, AFTER'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT outbox_id, record_changed_signal_id FROM workflow_ref
WHERE tenant_id = :'probe_tenant' AND record_id = :'probe_record' AND closed_at IS NULL AND record_changed_signal_id IS NOT NULL;

\echo '== sizes'
SELECT relname, pg_size_pretty(pg_relation_size(oid)) AS size, pg_relation_size(oid) AS bytes FROM pg_class
WHERE relname IN ('workflow_definition_versions', 'workflow_definition_versions_trigger', 'workflow_definition_versions_pkey',
                  'workflow_ref', 'workflow_ref_waiting', 'workflow_ref_record', 'workflow_ref_pkey') ORDER BY relname;
