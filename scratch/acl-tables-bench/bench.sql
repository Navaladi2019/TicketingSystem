-- P3-30 (TICK-615): record_shares list-join index, (tenant_id, principal_id)
-- vs (tenant_id, principal_id, record_id). Superuser, no RLS: this compares
-- index shapes only. 50 tenants; tenant 1 is the big one.
\timing off
SET max_parallel_workers_per_gather = 0;
CREATE TABLE records (tenant_id uuid, id uuid, updated_at timestamptz, PRIMARY KEY (tenant_id, id));
CREATE INDEX records_tenant_updated ON records (tenant_id, updated_at DESC, id DESC);
CREATE TABLE record_shares (tenant_id uuid NOT NULL, record_id uuid NOT NULL, principal_id uuid NOT NULL,
  principal_kind text NOT NULL, granted_by uuid, granted_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, record_id, principal_kind, principal_id));
-- 1,000,000 records: tenant 1 holds 400,000, the other 49 hold 12,245 each.
INSERT INTO records
SELECT ('00000000-0000-4000-8000-' || lpad(t::text, 12, '0'))::uuid, gen_random_uuid(), now() - (random() * interval '365 days')
FROM generate_series(1, 400000) g, LATERAL (SELECT 1 AS t) x;
INSERT INTO records
SELECT ('00000000-0000-4000-8000-' || lpad((2 + g % 49)::text, 12, '0'))::uuid, gen_random_uuid(), now() - (random() * interval '365 days')
FROM generate_series(1, 600000) g;
-- 2 shares per record on average: 2,000 principals per tenant (users and groups).
INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind)
SELECT r.tenant_id, r.id,
       ('00000000-0000-4000-9000-' || lpad((floor(random() * 2000))::int::text, 12, '0'))::uuid,
       (ARRAY['user','group'])[1 + (s % 2)]
FROM records r, generate_series(1, 2) s
ON CONFLICT DO NOTHING;
VACUUM ANALYZE records; VACUUM ANALYZE record_shares;
SELECT count(*) AS shares FROM record_shares;
SELECT version();

\echo '== A: (tenant_id, principal_id)'
CREATE INDEX record_shares_principal ON record_shares (tenant_id, principal_id);
VACUUM ANALYZE record_shares;
SELECT pg_size_pretty(pg_relation_size('record_shares_principal')) AS idx_size, pg_size_pretty(pg_relation_size('record_shares')) AS heap;
-- The caller is one user in 3 groups; tenant 1, newest 50 shared records.
PREPARE q AS
SELECT r.id, r.updated_at FROM records r
WHERE r.tenant_id = '00000000-0000-4000-8000-000000000001'
  AND r.id IN (SELECT s.record_id FROM record_shares s
               WHERE s.tenant_id = '00000000-0000-4000-8000-000000000001'
                 AND s.principal_id = ANY ('{00000000-0000-4000-9000-000000000007,00000000-0000-4000-9000-000000000011,00000000-0000-4000-9000-000000000013,00000000-0000-4000-9000-000000000017}'::uuid[]))
ORDER BY r.updated_at DESC, r.id DESC LIMIT 50;
\o /dev/null
EXECUTE q; EXECUTE q;
\o
EXPLAIN (ANALYZE, BUFFERS) EXECUTE q;
EXPLAIN (ANALYZE, BUFFERS) EXECUTE q;
DEALLOCATE q;

\echo '== B: (tenant_id, principal_id, record_id)'
DROP INDEX record_shares_principal;
CREATE INDEX record_shares_principal ON record_shares (tenant_id, principal_id, record_id);
VACUUM ANALYZE record_shares;
SELECT pg_size_pretty(pg_relation_size('record_shares_principal')) AS idx_size;
PREPARE q AS
SELECT r.id, r.updated_at FROM records r
WHERE r.tenant_id = '00000000-0000-4000-8000-000000000001'
  AND r.id IN (SELECT s.record_id FROM record_shares s
               WHERE s.tenant_id = '00000000-0000-4000-8000-000000000001'
                 AND s.principal_id = ANY ('{00000000-0000-4000-9000-000000000007,00000000-0000-4000-9000-000000000011,00000000-0000-4000-9000-000000000013,00000000-0000-4000-9000-000000000017}'::uuid[]))
ORDER BY r.updated_at DESC, r.id DESC LIMIT 50;
\o /dev/null
EXECUTE q; EXECUTE q;
\o
EXPLAIN (ANALYZE, BUFFERS) EXECUTE q;
EXPLAIN (ANALYZE, BUFFERS) EXECUTE q;

\echo '== insert cost: 20,000 shares, per index shape'
\timing on
INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind)
SELECT tenant_id, id, '00000000-0000-4000-9000-999999999999', 'user' FROM records WHERE tenant_id = '00000000-0000-4000-8000-000000000001' LIMIT 20000;
\timing off
DELETE FROM record_shares WHERE principal_id = '00000000-0000-4000-9000-999999999999';
DROP INDEX record_shares_principal;
CREATE INDEX record_shares_principal ON record_shares (tenant_id, principal_id);
VACUUM ANALYZE record_shares;
\timing on
INSERT INTO record_shares (tenant_id, record_id, principal_id, principal_kind)
SELECT tenant_id, id, '00000000-0000-4000-9000-999999999999', 'user' FROM records WHERE tenant_id = '00000000-0000-4000-8000-000000000001' LIMIT 20000;
