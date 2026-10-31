-- 800k records: tenant 0 has 400,000; tenants 1..399 share the other 400,000
-- (1,002 or 1,003 each). Fixed shape (TICK-360): 50 groups per tenant, 3
-- groups per user, 15% of rows open, ~1 KB of data per row, updated_at skewed
-- to recent rows, fixed RNG seed. Runs as the superuser (bypasses RLS).
\timing on
SET max_parallel_workers_per_gather = 0;  -- one RNG stream, so the seed reproduces
SELECT setseed(0.339);

CREATE TABLE bench_tenant AS
SELECT t AS tn,
       md5('tenant' || t)::uuid AS tenant_id,
       CASE WHEN t = 0 THEN 400000 ELSE 1002 + CASE WHEN t <= 202 THEN 1 ELSE 0 END END AS nrec,
       CASE WHEN t = 0 THEN 5000 ELSE 20 END AS nusers
FROM generate_series(0, 399) t;

-- requester: skewed (random()^2), so a few heavy requesters have > 1,000 rows
-- (page 20 of query b exists for them). assignee: uniform, 90% assigned.
-- group: uniform over the tenant's 50. priority: 1 = 5%, 2 = 15%, 3 = 50%,
-- 4 = 20%, 5 = 10%. Open state is independent of updated_at (the pessimistic
-- case for a walk down updated_at).
INSERT INTO records
SELECT x.tenant_id,
       md5('r' || x.tn || '/' || x.n)::uuid,
       md5('ed' || x.tn || '/' || (x.n % 5))::uuid,
       x.n,
       CASE WHEN x.r1 < 0.15 THEN (ARRAY['new', 'in_progress', 'on_hold'])[1 + floor(x.r2 * 3)::int] ELSE 'closed' END,
       md5('u' || x.tn || '/' || floor(x.nusers * x.r3 * x.r3)::int)::uuid,
       CASE WHEN x.r4 < 0.9 THEN md5('u' || x.tn || '/' || floor(x.nusers * x.r5)::int)::uuid END,
       md5('g' || x.tn || '/' || floor(50 * x.r6)::int)::uuid,
       x.upd - x.r7 * interval '30 days',
       x.upd,
       jsonb_build_object(
           'priority', CASE WHEN x.r8 < 0.05 THEN '1' WHEN x.r8 < 0.20 THEN '2' WHEN x.r8 < 0.70 THEN '3'
                            WHEN x.r8 < 0.90 THEN '4' ELSE '5' END,
           'category', 'category-' || floor(x.r9 * 20)::int,
           'impact', 1 + floor(x.r2 * 3)::int,
           'title', 'Record ' || x.n || ' ' || md5(x.n::text),
           'description', repeat(md5(x.tn || ':' || x.n), 27),
           'tags', jsonb_build_array('t' || floor(x.r5 * 50)::int, 't' || floor(x.r6 * 50)::int))
FROM (
    SELECT t.tn, t.tenant_id, t.nusers, n,
           random() AS r1, random() AS r2, random() AS r3, random() AS r4, random() AS r5,
           random() AS r6, random() AS r7, random() AS r8, random() AS r9,
           timestamptz '2026-10-01 00:00:00+00' - power(random(), 3) * interval '365 days' AS upd
    FROM bench_tenant t, generate_series(1, t.nrec) n
    ORDER BY t.tn, n
) x;

-- 2% of records shared with one random user of the tenant.
INSERT INTO record_shares
SELECT r.tenant_id, r.id, md5('u' || t.tn || '/' || floor(t.nusers * random())::int)::uuid
FROM records r JOIN bench_tenant t USING (tenant_id)
WHERE random() < 0.02
ON CONFLICT DO NOTHING;

-- Actors for the load: 200 users of the large tenant, one user per small
-- tenant. A user's 3 groups: g(u), g(u+17), g(u+34) mod 50.
CREATE TABLE bench_actor AS
SELECT 'large'::text AS kind, n, t.tenant_id, md5('u0/' || u)::uuid AS user_id,
       ARRAY[md5('g0/' || (u % 50))::uuid, md5('g0/' || ((u + 17) % 50))::uuid,
             md5('g0/' || ((u + 34) % 50))::uuid] AS groups
FROM bench_tenant t, generate_series(1, 200) n, LATERAL (SELECT (n * 24 + 7) % 5000 AS u) uu
WHERE t.tn = 0
UNION ALL
SELECT 'small', t.tn, t.tenant_id, md5('u' || t.tn || '/' || (t.tn % 20))::uuid,
       ARRAY[md5('g' || t.tn || '/' || (t.tn % 50))::uuid, md5('g' || t.tn || '/' || ((t.tn + 17) % 50))::uuid,
             md5('g' || t.tn || '/' || ((t.tn + 34) % 50))::uuid]
FROM bench_tenant t WHERE t.tn > 0;
-- Heavy requesters of the large tenant (> 1,000 records): query b, page 20.
CREATE TABLE bench_heavy AS
SELECT row_number() OVER (ORDER BY count(*) DESC, r.requester_id) AS n, r.tenant_id,
       r.requester_id AS user_id, count(*) AS nrows
FROM records r WHERE r.tenant_id = md5('tenant0')::uuid
GROUP BY r.tenant_id, r.requester_id HAVING count(*) > 1000;

VACUUM (ANALYZE) records;
VACUUM (ANALYZE) record_shares;
SELECT count(*) AS records, count(*) FILTER (WHERE state <> 'closed') AS open,
       count(*) FILTER (WHERE data ->> 'priority' = '1') AS priority_1,
       round(avg(pg_column_size(data))) AS avg_data_bytes FROM records;
SELECT (SELECT count(*) FROM record_shares) AS shares, (SELECT count(*) FROM bench_heavy) AS heavy_requesters;
