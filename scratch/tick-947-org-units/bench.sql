-- TICK-947 benchmark (docs/data/org-units.md). Run by bench.sh as the
-- superuser on a throwaway postgres:16.10. Shapes follow the real DDL
-- (db/migrations/20261121000000..0400); records is trimmed to the columns
-- that matter plus a 300-byte data payload. Owner and app roles stand in for
-- nava_migrator and nava_app, under FORCE RLS.
\set ON_ERROR_STOP on
\timing on
SET jit = off;
SET max_parallel_workers_per_gather = 0;

CREATE ROLE bench_owner NOLOGIN;
CREATE ROLE bench_app NOLOGIN;
GRANT USAGE, CREATE ON SCHEMA public TO bench_owner;
GRANT USAGE ON SCHEMA public TO bench_app;

CREATE TABLE org_units (
    tenant_id uuid NOT NULL, id uuid NOT NULL, parent_id uuid, name text NOT NULL,
    deleted_at timestamptz,
    live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED,
    CONSTRAINT org_units_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT org_units_live_key UNIQUE (tenant_id, id, live));
CREATE TABLE principals (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
CREATE TABLE records (
    tenant_id uuid NOT NULL, id uuid NOT NULL, entity_definition_id uuid NOT NULL,
    updated_at timestamptz NOT NULL, data jsonb NOT NULL,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id));
CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);

-- 400 tenants (tenant n = 0 is the big one), 200 units in each of the first
-- 10, 5,000 principals in tenant 0, 1,000,000 records: 400,000 in tenant 0,
-- the rest spread over the other 399.
CREATE TABLE t AS SELECT n, ('00000000-0000-4000-8000-' || lpad(to_hex(n), 12, '0'))::uuid id FROM generate_series(0, 399) n;
INSERT INTO org_units (tenant_id, id, name)
SELECT t.id, gen_random_uuid(), 'u' || u FROM t, generate_series(1, 200) u WHERE t.n < 10;
INSERT INTO principals SELECT t.id, gen_random_uuid() FROM t, generate_series(1, 5000) WHERE t.n = 0;
INSERT INTO records
SELECT CASE WHEN g <= 400000 THEN '00000000-0000-4000-8000-000000000000'::uuid
            ELSE ('00000000-0000-4000-8000-' || lpad(to_hex(1 + g % 399), 12, '0'))::uuid END,
       gen_random_uuid(), '00000000-0000-4000-8000-0000000000e1', now() - g * interval '1 second',
       jsonb_build_object('short_description', repeat('x', 300))
FROM generate_series(1, 1000000) g;
ALTER TABLE org_units OWNER TO bench_owner;
ALTER TABLE records OWNER TO bench_owner;
ALTER TABLE principals OWNER TO bench_owner;
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;
ALTER TABLE org_units ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_units FORCE ROW LEVEL SECURITY;
CREATE POLICY records_tenant ON records FOR ALL TO bench_app
    USING (tenant_id = current_setting('app.tenant_id')::uuid) WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY org_units_tenant ON org_units FOR SELECT TO bench_app USING (tenant_id = current_setting('app.tenant_id')::uuid);
GRANT SELECT, INSERT, UPDATE ON records TO bench_app;
GRANT SELECT ON org_units TO bench_app;
VACUUM ANALYZE;
SELECT pg_size_pretty(pg_total_relation_size('records')) records_total, count(*) FROM records;

\echo '== M3: ADD COLUMN + FK NOT VALID (catalogue only)'
SET ROLE bench_owner;
BEGIN;
ALTER TABLE records ADD COLUMN org_unit_id uuid,
    ADD CONSTRAINT records_org_unit_fkey FOREIGN KEY (tenant_id, org_unit_id) REFERENCES org_units (tenant_id, id) NOT VALID;
COMMIT;
\echo '== M4: VALIDATE with the TICK-715 migrator policies (every row NULL)'
BEGIN;
CREATE POLICY records_migrator_validate ON records FOR SELECT TO bench_owner USING (true);
CREATE POLICY org_units_migrator_validate ON org_units FOR SELECT TO bench_owner USING (true);
ALTER TABLE records VALIDATE CONSTRAINT records_org_unit_fkey;
DROP POLICY org_units_migrator_validate ON org_units;
DROP POLICY records_migrator_validate ON records;
COMMIT;
\echo '== M5: CREATE INDEX CONCURRENTLY, partial (every row NULL)'
CREATE INDEX CONCURRENTLY records_org_unit ON records (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL;
RESET ROLE;
SELECT pg_size_pretty(pg_relation_size('records_org_unit')) idx_all_null;

\echo '== Tenant 0 adopts regions: its 400,000 rows each get one of its 200 units'
UPDATE records r SET org_unit_id = u.id
FROM (SELECT id, row_number() OVER (ORDER BY id) - 1 k FROM org_units WHERE tenant_id = '00000000-0000-4000-8000-000000000000') u
WHERE r.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.k = (hashtext(r.id::text) & 2147483647) % 200;
VACUUM ANALYZE records;
REINDEX INDEX records_org_unit;
SELECT pg_size_pretty(pg_relation_size('records_org_unit')) idx_400k_rows,
       pg_size_pretty(pg_relation_size('records_pkey')) pkey_1m_rows_for_scale,
       pg_size_pretty(pg_relation_size('records')) heap;

\echo '== Read as bench_app, tenant 0: page 1 of a 20-unit subtree (set S); count of one unit'
SET ROLE bench_app;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000000000', false);
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id FROM records
WHERE org_unit_id IS NOT NULL AND org_unit_id = ANY ((SELECT array_agg(id) FROM (SELECT id FROM org_units ORDER BY id LIMIT 20) x)::uuid[])
ORDER BY updated_at DESC, id DESC LIMIT 51;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM records
WHERE org_unit_id IS NOT NULL AND org_unit_id = ANY ((SELECT array_agg(id) FROM (SELECT id FROM org_units ORDER BY id LIMIT 1) x)::uuid[]);
RESET ROLE;
\echo '== The same two reads without records_org_unit (before)'
DROP INDEX records_org_unit;
SET ROLE bench_app;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id FROM records
WHERE org_unit_id IS NOT NULL AND org_unit_id = ANY ((SELECT array_agg(id) FROM (SELECT id FROM org_units ORDER BY id LIMIT 20) x)::uuid[])
ORDER BY updated_at DESC, id DESC LIMIT 51;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM records
WHERE org_unit_id IS NOT NULL AND org_unit_id = ANY ((SELECT array_agg(id) FROM (SELECT id FROM org_units ORDER BY id LIMIT 1) x)::uuid[]);
RESET ROLE;

\echo '== Write cost: 20,000 single-row inserts as bench_app, without then with the index'
CREATE FUNCTION ins(n int, unit uuid) RETURNS numeric LANGUAGE plpgsql AS $$
DECLARE t0 timestamptz := clock_timestamp();
BEGIN
  FOR i IN 1..n LOOP
    INSERT INTO records (tenant_id, id, entity_definition_id, updated_at, data, org_unit_id)
    VALUES (current_setting('app.tenant_id')::uuid, gen_random_uuid(), '00000000-0000-4000-8000-0000000000e1', now(), '{}', unit);
  END LOOP;
  RETURN round(extract(epoch FROM clock_timestamp() - t0) * 1e6 / n, 1);
END $$;
GRANT EXECUTE ON FUNCTION ins(int, uuid) TO bench_app;
SET ROLE bench_app;
SELECT ins(20000, NULL) us_per_insert_null_no_index;
SELECT ins(20000, (SELECT id FROM org_units ORDER BY id LIMIT 1)) us_per_insert_unit_no_index;
RESET ROLE;
CREATE INDEX records_org_unit ON records (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL;
SET ROLE bench_app;
SELECT ins(20000, NULL) us_per_insert_null_with_index;
SELECT ins(20000, (SELECT id FROM org_units ORDER BY id LIMIT 1)) us_per_insert_unit_with_index;
RESET ROLE;

\echo '== org_unit_members: tenant 0, 5,000 principals with 1-3 units each'
CREATE TABLE org_unit_members (
    tenant_id uuid NOT NULL, org_unit_id uuid NOT NULL, principal_id uuid NOT NULL,
    is_primary boolean NOT NULL DEFAULT false, live boolean NOT NULL DEFAULT true,
    CONSTRAINT org_unit_members_pkey PRIMARY KEY (tenant_id, org_unit_id, principal_id), CHECK (live),
    FOREIGN KEY (tenant_id, org_unit_id, live) REFERENCES org_units (tenant_id, id, live),
    FOREIGN KEY (tenant_id, principal_id) REFERENCES principals (tenant_id, id) ON DELETE CASCADE);
CREATE INDEX org_unit_members_principal ON org_unit_members (tenant_id, principal_id, org_unit_id);
CREATE UNIQUE INDEX org_unit_members_one_primary ON org_unit_members (tenant_id, principal_id) WHERE is_primary;
INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, is_primary)
SELECT p.tenant_id, u.id, p.id, k = 0
FROM (SELECT tenant_id, id, row_number() OVER (ORDER BY id) n FROM principals) p
CROSS JOIN LATERAL generate_series(0, (p.n % 3)::int) k
JOIN (SELECT id, row_number() OVER (ORDER BY id) - 1 j FROM org_units WHERE tenant_id = '00000000-0000-4000-8000-000000000000') u
  ON u.j = (p.n * 7 + k * 13) % 200;
VACUUM ANALYZE org_unit_members;
SELECT count(*) memberships, pg_size_pretty(pg_relation_size('org_unit_members_pkey')) pkey,
       pg_size_pretty(pg_relation_size('org_unit_members_principal')) principal_idx,
       pg_size_pretty(pg_relation_size('org_unit_members_one_primary')) one_primary
FROM org_unit_members;
\echo '== Units of a principal (the set S seed)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT org_unit_id FROM org_unit_members
WHERE tenant_id = '00000000-0000-4000-8000-000000000000'
  AND principal_id = (SELECT id FROM principals ORDER BY id LIMIT 1 OFFSET 2500);
\echo '== Tombstoning a unit that has members: the NO ACTION check refuses it via the pkey'
SELECT id AS busy FROM org_units WHERE id IN (SELECT org_unit_id FROM org_unit_members) LIMIT 1 \gset
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT 1 FROM org_unit_members
WHERE tenant_id = '00000000-0000-4000-8000-000000000000' AND org_unit_id = :'busy' AND live FOR KEY SHARE;
SELECT version();
