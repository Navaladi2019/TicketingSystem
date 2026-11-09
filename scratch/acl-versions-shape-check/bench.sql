-- TICK-629: run by run.sh. 2,000 tenants x 10 types x 5 versions = 100,000
-- ACL versions, each a realistic policy: 6 rules (2 groups and 10 users, the
-- requester / assignee / shared flags, a state IN and a my_groups condition)
-- and 3 field masks.
\pset pager off
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE acl_versions (
    tenant_id            uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    version              integer     NOT NULL,
    rules                jsonb       NOT NULL,
    masks                jsonb       NOT NULL,
    published_by         uuid,
    published_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, entity_definition_id, version)
);

CREATE FUNCTION bench_subject(n int) RETURNS jsonb LANGUAGE sql VOLATILE RETURN jsonb_build_object(
    'groups', (SELECT jsonb_agg(gen_random_uuid()) FROM generate_series(1, 2)),
    'users', (SELECT jsonb_agg(gen_random_uuid()) FROM generate_series(1, n)),
    'requester', true, 'shared', true);
CREATE FUNCTION bench_rules() RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(jsonb_build_object(
        'id', 'rule-' || i,
        'ops', '["read", "write"]'::jsonb,
        'who', bench_subject(10),
        'where', jsonb_build_array(
            jsonb_build_object('column', 'state', 'op', 'in', 'values', '["new", "open", "pending"]'::jsonb),
            jsonb_build_object('column', 'assignment_group_id', 'op', 'in', 'caller', 'my_groups'))))
    FROM generate_series(1, 6) i);
CREATE FUNCTION bench_masks() RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(jsonb_build_object('field', 'f' || i, 'read', bench_subject(0) - 'users', 'write', '{}'::jsonb))
    FROM generate_series(1, 3) i);

-- One pool of 20,000 policies, reused for the existing rows and the insert runs.
CREATE TABLE pool AS SELECT i, bench_rules() AS rules, bench_masks() AS masks FROM generate_series(1, 20000) i;
CREATE UNIQUE INDEX ON pool (i);

INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
SELECT t.id, d.id, v, p.rules, p.masks
FROM (SELECT gen_random_uuid() id, row_number() OVER () n FROM generate_series(1, 2000)) t
CROSS JOIN LATERAL (SELECT gen_random_uuid() id, row_number() OVER () n FROM generate_series(1, 10)) d
CROSS JOIN generate_series(1, 5) v
JOIN pool p ON p.i = 1 + ((t.n * 50 + d.n * 5 + v) % 20000);
VACUUM ANALYZE acl_versions;

SELECT count(*) AS acl_versions_rows,
       pg_size_pretty(pg_total_relation_size('acl_versions')) AS total_size,
       round(avg(pg_column_size(rules))) AS avg_rules_bytes,
       round(avg(pg_column_size(masks))) AS avg_masks_bytes,
       round(avg(octet_length(rules::text))) AS avg_rules_text_bytes
FROM acl_versions;

-- Insert target with the same columns, no CHECK yet; truncated between runs.
CREATE TABLE ins (LIKE acl_versions INCLUDING ALL);

\timing on
\echo '== insert 20,000 versions, no CHECK (3 runs)'
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool;
TRUNCATE ins;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool;
TRUNCATE ins;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool;
TRUNCATE ins;
\echo '== one publish (single-row insert), no CHECK (3 runs)'
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool WHERE i = 1;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool WHERE i = 2;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool WHERE i = 3;
TRUNCATE ins;

\echo '== migration 1: functions + ADD CONSTRAINT ... NOT VALID (100,000 rows present)'
\i /bench/m1.sql
\echo '== migration 2: VALIDATE CONSTRAINT x2 over 100,000 rows'
\i /bench/m2.sql

ALTER TABLE ins ADD CONSTRAINT ins_rules_check CHECK (acl_rules_valid(rules)),
                ADD CONSTRAINT ins_masks_check CHECK (acl_masks_valid(masks));
\echo '== insert 20,000 versions, both CHECKs (3 runs)'
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool;
TRUNCATE ins;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool;
TRUNCATE ins;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool;
TRUNCATE ins;

\echo '== one publish (single-row insert), both CHECKs (3 runs)'
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool WHERE i = 1;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool WHERE i = 2;
INSERT INTO ins (tenant_id, entity_definition_id, version, rules, masks) SELECT gen_random_uuid(), gen_random_uuid(), 1, rules, masks FROM pool WHERE i = 3;
TRUNCATE ins;
\timing off

\echo '== both predicates alone over 100,000 rows'
SET max_parallel_workers_per_gather = 0;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT count(*) FROM acl_versions WHERE acl_rules_valid(rules) AND acl_masks_valid(masks);
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT count(*) FROM acl_versions;

\echo '== a malformed row is refused, naming the constraint'
\set ON_ERROR_STOP 0
INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
VALUES (gen_random_uuid(), gen_random_uuid(), 1, '[{"id": "x", "ops": ["read"], "who": {}, "colour": "red"}]', '[]');
INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
VALUES (gen_random_uuid(), gen_random_uuid(), 1, '[]', '{"fields": []}');
