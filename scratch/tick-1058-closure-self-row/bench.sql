-- TICK-1058 benchmark (docs/data/org-unit-closure.md, "Unrooted-unit check").
-- Lines 1-71 are scratch/tick-1032-org-unit-closure/bench.sql's setup and
-- backfill (400 tenants, 13,310 units; tenant 0: 11,111 units, one level-2
-- subtree of 111 tombstoned), then the check group sync runs after each
-- rebuild: internal/groupsync/closure.go closureUnrootedSQL.
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
ANALYZE org_unit_closure;

\echo '== check, tenant 0 (110: the setup tombstones a level-2 unit but not its 110 live descendants, a state group sync never leaves)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM org_units u
WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
    WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);
SELECT count(*) AS unrooted_tenant0 FROM org_units u
WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
    WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);

\echo '== check, tenant 1 (201 units)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM org_units u
WHERE u.tenant_id = '00000000-0000-4000-8000-000000000001' AND u.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
    WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);

\echo '== one orphan in tenant 0 (FK triggers off): 111'
SET session_replication_role = replica;
INSERT INTO org_units (tenant_id, id, parent_id, name)
VALUES ('00000000-0000-4000-8000-000000000000', gen_random_uuid(), gen_random_uuid(), 'orphan');
RESET session_replication_role;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM org_units u
WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
    WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);
SELECT count(*) AS unrooted_tenant0 FROM org_units u
WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
    WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);

\echo '== the same with hash join off (the PK-probe nested loop the planner rejected)'
SET enable_hashjoin = off;
SET enable_mergejoin = off;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT count(*) FROM org_units u
WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.deleted_at IS NULL
AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
    WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);
RESET enable_hashjoin;
RESET enable_mergejoin;

\echo '== tenant 0 rebuild for scale (DELETE then INSERT, the group sync statement)'
BEGIN;
DELETE FROM org_unit_closure WHERE tenant_id = '00000000-0000-4000-8000-000000000000';
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
WITH RECURSIVE live (id) AS (
    SELECT u.id FROM org_units u WHERE u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.parent_id IS NULL AND u.deleted_at IS NULL
    UNION ALL
    SELECT u.id FROM live l JOIN org_units u ON u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.parent_id = l.id WHERE u.deleted_at IS NULL
), closure (ancestor_id, descendant_id) AS (
    SELECT id, id FROM live
    UNION ALL
    SELECT c.ancestor_id, u.id FROM closure c JOIN org_units u ON u.tenant_id = '00000000-0000-4000-8000-000000000000' AND u.parent_id = c.descendant_id WHERE u.deleted_at IS NULL
) SELECT '00000000-0000-4000-8000-000000000000', ancestor_id, descendant_id FROM closure;
COMMIT;
