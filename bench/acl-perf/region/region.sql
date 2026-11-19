-- SP-1 (TICK-932, PLAN-011 P11-01): bench-only region data on the P3-36
-- 800k seed. NOT a migration: records.org_unit_id is S31's job (TICK-947).
-- Runs as the superuser in the kept bench database `nava`.
--
-- Tree (large tenant only, tn = 0): root -> 10 regions -> 10 districts each
-- -> 10 areas each -> 9 branches each = 1 + 10 + 100 + 1,000 + 9,000 = 10,111
-- units. Subtree sizes: area 10, district 101, region 1,011, root 10,111.
-- `pre` is the pre-order (depth-first) position, so every subtree is the
-- contiguous range [pre, pre + size).
--
-- People sit on branches (leaves): principal u's primary unit is branch
-- md5-hash(u) mod 9,000; 5% of principals (hash mod 20 = 0) have none.
-- Records take the requester's primary unit (ADR-010 D8), so the record
-- distribution over units follows the seed's requester skew (r3^2).
\set ON_ERROR_STOP on
SET session_replication_role = replica;

DROP TABLE IF EXISTS bench_ou;
CREATE TABLE bench_ou (
    pre     int PRIMARY KEY,
    id      uuid NOT NULL,
    parent  uuid,
    depth   int NOT NULL,
    leaf_no int
);
INSERT INTO bench_ou (pre, id, parent, depth, leaf_no)
SELECT 0, md5('ou/root')::uuid, NULL::uuid, 0, NULL::int
UNION ALL
SELECT 1 + r * 1011, md5('ou/' || r)::uuid, md5('ou/root')::uuid, 1, NULL::int
FROM generate_series(0, 9) r
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101, md5('ou/' || r || '/' || d)::uuid, md5('ou/' || r)::uuid, 2, NULL
FROM generate_series(0, 9) r, generate_series(0, 9) d
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101 + 1 + a * 10, md5('ou/' || r || '/' || d || '/' || a)::uuid,
       md5('ou/' || r || '/' || d)::uuid, 3, NULL
FROM generate_series(0, 9) r, generate_series(0, 9) d, generate_series(0, 9) a
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101 + 1 + a * 10 + 1 + b, md5('ou/' || r || '/' || d || '/' || a || '/' || b)::uuid,
       md5('ou/' || r || '/' || d || '/' || a)::uuid, 4, ((r * 10 + d) * 10 + a) * 9 + b
FROM generate_series(0, 9) r, generate_series(0, 9) d, generate_series(0, 9) a, generate_series(0, 8) b;

DELETE FROM org_units WHERE tenant_id = (SELECT tenant_id FROM perf_tenant WHERE tn = 0);
INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
SELECT t.tenant_id, o.id, o.parent, 'bench-ou-' || o.pre, 'ou-' || o.pre
FROM bench_ou o, perf_tenant t WHERE t.tn = 0 ORDER BY o.pre;

-- Primary unit per large-tenant principal.
DROP TABLE IF EXISTS bench_primary;
CREATE TABLE bench_primary AS
SELECT md5('u0/' || u)::uuid AS principal_id,
       CASE WHEN (('x' || substr(md5('oum' || u), 1, 7))::bit(28)::int) % 20 = 0 THEN NULL
            ELSE (SELECT id FROM bench_ou WHERE leaf_no = (('x' || substr(md5('ou' || u), 1, 7))::bit(28)::int) % 9000)
       END AS org_unit_id
FROM generate_series(0, 4999) u;

ALTER TABLE records ADD COLUMN IF NOT EXISTS org_unit_id uuid;
UPDATE records r SET org_unit_id = p.org_unit_id
FROM bench_primary p, perf_tenant t
WHERE t.tn = 0 AND r.tenant_id = t.tenant_id AND r.requester_id = p.principal_id;
SET session_replication_role = origin;

-- The UPDATE wrote a new version of every large-tenant row. VACUUM FULL
-- rewrites the heap in its existing order and rebuilds every index, so the
-- baseline and region rows below run on one compact table.
VACUUM FULL records;
VACUUM (ANALYZE) records;
ANALYZE org_units;
CHECKPOINT;
