-- T18a (TICK-194) benchmark: tenant_email_domain lookups as nava_app under
-- FORCE RLS. Run as the superuser on a migrated `nava` database
-- (a dbtest cluster), then the EXPLAIN part as nava_app:
--   psql -U pgadmin -d nava -f scratch/t18a_email_domain_bench.sql
-- Volume: 10,000 active tenants with 1-3 domains each (~20,000 rows) plus one
-- large tenant with 5,000 domains (skew).
\set ON_ERROR_STOP on
TRUNCATE tenant_email_domain;
DELETE FROM tenants WHERE id NOT IN (SELECT tenant_id FROM tenant_identity_binding);
INSERT INTO tenants (id, state)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'active'
FROM generate_series(1, 10001) g ON CONFLICT DO NOTHING;
INSERT INTO tenant_email_domain (tenant_id, domain)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 't' || g || '-d' || d || '.example'
FROM generate_series(1, 10000) g, generate_series(1, 1 + g % 3) d;
INSERT INTO tenant_email_domain (tenant_id, domain)
SELECT '00000000-0000-4000-8000-000000002711', 'big-d' || d || '.example' FROM generate_series(1, 5000) d;
VACUUM ANALYZE tenant_email_domain;
SELECT count(*) AS rows, pg_size_pretty(pg_relation_size('tenant_email_domain')) AS heap,
       pg_size_pretty(pg_relation_size('tenant_email_domain_pkey')) AS pkey
FROM tenant_email_domain;

SET ROLE nava_app;
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000000005', true);
-- owned
EXPLAIN (ANALYZE, BUFFERS) SELECT EXISTS (SELECT 1 FROM tenant_email_domain
WHERE tenant_id = '00000000-0000-4000-8000-000000000005' AND domain = 't5-d2.example');
-- another tenant's (the large tenant's) domain: reads as not owned
EXPLAIN (ANALYZE, BUFFERS) SELECT EXISTS (SELECT 1 FROM tenant_email_domain
WHERE tenant_id = '00000000-0000-4000-8000-000000000005' AND domain = 'big-d17.example');
-- nobody's
EXPLAIN (ANALYZE, BUFFERS) SELECT EXISTS (SELECT 1 FROM tenant_email_domain
WHERE tenant_id = '00000000-0000-4000-8000-000000000005' AND domain = 'nobody.example');
COMMIT;
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000002711', true);
-- the large tenant, owned
EXPLAIN (ANALYZE, BUFFERS) SELECT EXISTS (SELECT 1 FROM tenant_email_domain
WHERE tenant_id = '00000000-0000-4000-8000-000000002711' AND domain = 'big-d4999.example');
COMMIT;
RESET ROLE;
