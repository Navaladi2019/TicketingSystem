-- TICK-671 (P6-06): the number-format tables, the epoch registry and the
-- pinned link at realistic row counts, read as an RLS-bound role with the
-- canonical predicate, as nava_app reads them. Shapes copy
-- db/migrations/20261113100000 and 20261113100100 (without the principals and
-- tenants FKs, which the queries below never touch).
\timing off
SET client_min_messages = warning;
DROP ROLE IF EXISTS bench_app;
CREATE ROLE bench_app;
CREATE TABLE entity_definitions (tenant_id uuid, id uuid, key text, PRIMARY KEY (tenant_id, id), UNIQUE (tenant_id, key));
CREATE TABLE number_formats (
    tenant_id uuid NOT NULL, id uuid NOT NULL, key text NOT NULL, current_version int NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT number_formats_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT number_formats_key_key UNIQUE (tenant_id, key));
CREATE TABLE number_format_versions (
    tenant_id uuid NOT NULL, format_id uuid NOT NULL, version int NOT NULL, document jsonb NOT NULL,
    compiled jsonb NOT NULL, compiled_sha256 bytea NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), created_by uuid NOT NULL,
    CONSTRAINT number_format_versions_pkey PRIMARY KEY (tenant_id, format_id, version),
    FOREIGN KEY (tenant_id, format_id) REFERENCES number_formats (tenant_id, id));
CREATE TABLE numbering_epochs (
    tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL, epoch int NOT NULL, compiled_sha256 bytea NOT NULL,
    literal_head text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT numbering_epochs_pkey PRIMARY KEY (tenant_id, entity_definition_id, epoch),
    CONSTRAINT numbering_epochs_compiled_sha256_key UNIQUE (tenant_id, entity_definition_id, compiled_sha256),
    FOREIGN KEY (tenant_id, entity_definition_id) REFERENCES entity_definitions (tenant_id, id));
CREATE TABLE entity_definition_versions (
    tenant_id uuid NOT NULL, definition_id uuid NOT NULL, version int NOT NULL, schema json NOT NULL,
    PRIMARY KEY (tenant_id, definition_id, version));

-- 2,000 tenants x 50 types x 5 versions; 10 formats x 3 versions per tenant;
-- 2 epochs per type; each type's version 5 pinned.
INSERT INTO entity_definitions
SELECT ('00000000-0000-4000-8000-' || lpad(t::text, 12, '0'))::uuid, gen_random_uuid(), 'type_' || d
FROM generate_series(1, 2000) t, generate_series(1, 50) d;
INSERT INTO number_formats (tenant_id, id, key, current_version)
SELECT ('00000000-0000-4000-8000-' || lpad(t::text, 12, '0'))::uuid, gen_random_uuid(), 'fmt_' || f, 3
FROM generate_series(1, 2000) t, generate_series(1, 10) f;
INSERT INTO number_format_versions (tenant_id, format_id, version, document, compiled, compiled_sha256, created_by)
SELECT f.tenant_id, f.id, v, '{"template": "{prefix}-{yyyy}{MM}{seq:4}", "overflow": "refuse"}',
       '{"scope_parts": ["d:yyyyMM"], "head": "-", "width": 4, "tail": "", "overflow": "refuse", "epoch": 1, "time_zone": "Asia/Kolkata"}',
       sha256((f.id::text || v)::bytea), f.tenant_id
FROM number_formats f, generate_series(1, 3) v;
INSERT INTO numbering_epochs (tenant_id, entity_definition_id, epoch, compiled_sha256, literal_head)
SELECT d.tenant_id, d.id, e, sha256((d.id::text || e)::bytea), 'TCK-'
FROM entity_definitions d, generate_series(1, 2) e;
INSERT INTO entity_definition_versions SELECT d.tenant_id, d.id, v, '{"type": "object"}'
FROM entity_definitions d, generate_series(1, 5) v;
VACUUM ANALYZE;

\echo '== migration 20261113100100 / 100200 on the filled entity_definition_versions (500,000 rows)'
\timing on
ALTER TABLE entity_definition_versions
    ADD COLUMN number_format_id uuid, ADD COLUMN number_format_version integer, ADD COLUMN number_compiled jsonb,
    ADD CONSTRAINT entity_definition_versions_number_format_fkey
        FOREIGN KEY (tenant_id, number_format_id, number_format_version)
        REFERENCES number_format_versions (tenant_id, format_id, version);
ALTER TABLE entity_definition_versions ADD CONSTRAINT edv_nf_check CHECK (
    (number_format_id IS NULL) = (number_format_version IS NULL) AND (number_format_id IS NULL) = (number_compiled IS NULL)) NOT VALID;
ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT edv_nf_check;
\timing off

UPDATE entity_definition_versions v SET number_format_id = f.id, number_format_version = 3, number_compiled = '{"head": "-"}'
FROM number_formats f
WHERE v.version = 5 AND f.tenant_id = v.tenant_id AND f.key = 'fmt_' || (1 + abs(hashtext(v.definition_id::text)) % 10);
VACUUM ANALYZE;

DO $$ DECLARE t text; BEGIN
  FOREACH t IN ARRAY ARRAY['entity_definitions', 'number_formats', 'number_format_versions', 'numbering_epochs', 'entity_definition_versions'] LOOP
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
    EXECUTE format('CREATE POLICY p ON %I FOR ALL TO bench_app USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting(''app.tenant_id'', true), '''')::uuid)', t);
    EXECUTE format('GRANT SELECT, INSERT, UPDATE ON %I TO bench_app', t);
  END LOOP;
END $$;

SELECT relname, n_live_tup, pg_size_pretty(pg_table_size(relid)) AS heap, pg_size_pretty(pg_indexes_size(relid)) AS indexes
FROM pg_stat_user_tables ORDER BY relname;

SET ROLE bench_app;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000001234', false);
\echo '== (a) format by key (admin GET, publish)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id, current_version FROM number_formats WHERE key = 'fmt_7';
\echo '== (b) a pinned version by key and version (link validation at definition publish)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT v.compiled, v.compiled_sha256 FROM number_format_versions v
  JOIN number_formats f ON f.tenant_id = v.tenant_id AND f.id = v.format_id WHERE f.key = 'fmt_7' AND v.version = 3;
\echo '== (c) every epoch of a type (disjointness check, C10)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT e.epoch, e.literal_head, e.compiled_sha256 FROM numbering_epochs e
  JOIN entity_definitions d ON d.tenant_id = e.tenant_id AND d.id = e.entity_definition_id WHERE d.key = 'type_9';
\echo '== (d) the epoch of one compiled template (re-adopt)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT e.epoch FROM numbering_epochs e
  JOIN entity_definitions d ON d.tenant_id = e.tenant_id AND d.id = e.entity_definition_id
  WHERE d.key = 'type_9' AND e.compiled_sha256 = sha256('x'::bytea);
\echo '== (e) types whose versions pin a format ("re-publish linked types"); no index on the link'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT DISTINCT v.definition_id FROM entity_definition_versions v
  JOIN number_formats f ON f.tenant_id = v.tenant_id AND f.id = v.number_format_id WHERE f.key = 'fmt_7';
\echo '== (f) a pinned definition-version insert: FK probe of number_format_versions_pkey'
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) INSERT INTO entity_definition_versions
  SELECT d.tenant_id, d.id, 6, '{}', f.id, 3, '{}' FROM entity_definitions d JOIN number_formats f ON f.tenant_id = d.tenant_id
  WHERE d.key = 'type_9' AND f.key = 'fmt_7';
ROLLBACK;
RESET ROLE;
