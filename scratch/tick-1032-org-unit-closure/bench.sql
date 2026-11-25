-- TICK-1032 benchmark (docs/data/org-unit-closure.md). Run by bench.sh as the
-- superuser on a throwaway postgres:16.10. org_units and org_unit_closure
-- follow the real DDL (db/migrations/20261121050000_org_unit_closure.sql);
-- bench_app stands in for nava_app under FORCE RLS.
\set ON_ERROR_STOP on
\timing on
SET jit = off;
SET max_parallel_workers_per_gather = 0;
SELECT version();

CREATE ROLE bench_app NOLOGIN;
GRANT USAGE ON SCHEMA public TO bench_app;

CREATE TABLE org_units (
    tenant_id uuid NOT NULL, id uuid NOT NULL, parent_id uuid, name text NOT NULL,
    deleted_at timestamptz,
    CONSTRAINT org_units_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT org_units_parent_fkey FOREIGN KEY (tenant_id, parent_id) REFERENCES org_units (tenant_id, id));
CREATE INDEX org_units_parent ON org_units (tenant_id, parent_id);
CREATE TABLE org_unit_closure (
    tenant_id uuid NOT NULL, ancestor_id uuid NOT NULL, descendant_id uuid NOT NULL,
    CONSTRAINT org_unit_closure_pkey PRIMARY KEY (tenant_id, ancestor_id, descendant_id),
    CONSTRAINT org_unit_closure_ancestor_fkey FOREIGN KEY (tenant_id, ancestor_id) REFERENCES org_units (tenant_id, id),
    CONSTRAINT org_unit_closure_descendant_fkey FOREIGN KEY (tenant_id, descendant_id) REFERENCES org_units (tenant_id, id));
ALTER TABLE org_unit_closure ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_unit_closure FORCE ROW LEVEL SECURITY;
CREATE POLICY org_unit_closure_select ON org_unit_closure FOR SELECT TO bench_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT ON org_unit_closure TO bench_app;

-- 400 tenants. Tenant 0 has a 5-level tree of 11,111 units, fan-out 10
-- (root, 10, 100, 1,000, 10,000); tenants 1-9 have 201 units each (root
-- plus 200 children); the other 390 have one root. One level-2 unit of
-- tenant 0 is tombstoned (its subtree: 111 units).
CREATE TABLE t AS SELECT n, ('00000000-0000-4000-8000-' || lpad(to_hex(n), 12, '0'))::uuid id FROM generate_series(0, 399) n;
CREATE TABLE ids AS
WITH RECURSIVE g (lvl, k, parent_k) AS (
    SELECT 0, 0::bigint, NULL::bigint
    UNION ALL
    SELECT g.lvl + 1, g.k * 10 + c, g.k FROM g, generate_series(1, 10) c WHERE g.lvl < 4)
SELECT lvl, k, parent_k, gen_random_uuid() id FROM g;
INSERT INTO org_units (tenant_id, id, parent_id, name)
SELECT '00000000-0000-4000-8000-000000000000', i.id, p.id, 'u' || i.k
FROM ids i LEFT JOIN ids p ON p.k = i.parent_k AND p.lvl = i.lvl - 1 ORDER BY i.lvl;
INSERT INTO org_units (tenant_id, id, name) SELECT t.id, t.id, 'root' FROM t WHERE t.n > 0;
INSERT INTO org_units (tenant_id, id, parent_id, name)
SELECT t.id, gen_random_uuid(), t.id, 'c' || c FROM t, generate_series(1, 200) c WHERE t.n BETWEEN 1 AND 9;
UPDATE org_units SET deleted_at = now()
WHERE tenant_id = '00000000-0000-4000-8000-000000000000' AND id = (SELECT id FROM ids WHERE lvl = 2 AND k = 11);
ANALYZE org_units;
SELECT count(*) AS org_units_rows FROM org_units;

\echo '== backfill (all 400 tenants, the migration statement per tenant)'
DO $$
DECLARE tid uuid;
BEGIN
    FOR tid IN SELECT id FROM t ORDER BY id LOOP
        INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
        WITH RECURSIVE live (id) AS (
            SELECT u.id FROM org_units u WHERE u.tenant_id = tid AND u.parent_id IS NULL AND u.deleted_at IS NULL
            UNION ALL
            SELECT u.id FROM live l JOIN org_units u ON u.tenant_id = tid AND u.parent_id = l.id WHERE u.deleted_at IS NULL
        ), closure (ancestor_id, descendant_id) AS (
            SELECT id, id FROM live
            UNION ALL
            SELECT c.ancestor_id, u.id FROM closure c JOIN org_units u ON u.tenant_id = tid AND u.parent_id = c.descendant_id
            WHERE u.deleted_at IS NULL
        )
        SELECT tid, ancestor_id, descendant_id FROM closure;
    END LOOP;
END $$;
VACUUM ANALYZE org_unit_closure;
SELECT count(*) AS closure_rows,
       count(*) FILTER (WHERE tenant_id = '00000000-0000-4000-8000-000000000000') AS tenant0_rows
FROM org_unit_closure;
SELECT pg_size_pretty(pg_relation_size('org_unit_closure')) AS heap,
       pg_size_pretty(pg_relation_size('org_unit_closure_pkey')) AS pkey,
       pg_size_pretty(pg_total_relation_size('org_unit_closure')) AS total;

\echo '== rebuild of tenant 0 (group sync: delete the tenant rows, insert them again)'
BEGIN;
DELETE FROM org_unit_closure WHERE tenant_id = '00000000-0000-4000-8000-000000000000';
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
WITH RECURSIVE live (id) AS (
    SELECT u.id FROM org_units u WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.parent_id IS NULL AND u.deleted_at IS NULL
    UNION ALL
    SELECT u.id FROM live l JOIN org_units u ON u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.parent_id = l.id WHERE u.deleted_at IS NULL
), closure (ancestor_id, descendant_id) AS (
    SELECT id, id FROM live
    UNION ALL
    SELECT c.ancestor_id, u.id FROM closure c JOIN org_units u ON u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.parent_id = c.descendant_id
    WHERE u.deleted_at IS NULL
)
SELECT '00000000-0000-4000-8000-000000000000', ancestor_id, descendant_id FROM closure;
COMMIT;
VACUUM ANALYZE org_unit_closure;

SELECT (SELECT id FROM ids WHERE lvl = 0) AS root, (SELECT id FROM ids WHERE lvl = 1 AND k = 1) AS l1,
       (SELECT id FROM ids WHERE lvl = 4 AND k = 5432) AS leaf \gset
SET ROLE bench_app;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000000000', false);
\echo '== Allows probe: is a level-4 leaf under the root?'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT EXISTS (SELECT 1 FROM org_unit_closure
    WHERE tenant_id = '00000000-0000-4000-8000-000000000000' AND ancestor_id = :'root' AND descendant_id = :'leaf');
\echo '== list leaf: the live subtree of the root (11,000 units)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT descendant_id FROM org_unit_closure
WHERE tenant_id = '00000000-0000-4000-8000-000000000000' AND ancestor_id = :'root';
\echo '== list leaf: the live subtree of a level-1 unit (1,000 live of 1,111)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT descendant_id FROM org_unit_closure
WHERE tenant_id = '00000000-0000-4000-8000-000000000000' AND ancestor_id = :'l1';
\echo '== the same, with only the RLS predicate on tenant_id'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT descendant_id FROM org_unit_closure WHERE ancestor_id = :'l1';
RESET ROLE;
