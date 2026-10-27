-- 10,000 active tenants with 1-3 domains each (20,000 rows), one large
-- active tenant with 5,000 domains, and one provisioning tenant with 3:
-- 25,003 rows, the volume of docs/data/identity.md's T18a evidence.
INSERT INTO tenants (id, state)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'active'
FROM generate_series(1, 10001) g;
INSERT INTO tenants (id, state) VALUES ('00000000-0000-4000-8000-0000000fffff', 'provisioning');
INSERT INTO tenant_email_domain (tenant_id, domain)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 't' || g || '-d' || d || '.example'
FROM generate_series(1, 10000) g, generate_series(1, 1 + g % 3) d;
INSERT INTO tenant_email_domain (tenant_id, domain)
SELECT '00000000-0000-4000-8000-000000002711', 'big-d' || d || '.example'
FROM generate_series(1, 5000) d;
INSERT INTO tenant_email_domain (tenant_id, domain)
SELECT '00000000-0000-4000-8000-0000000fffff', 'stuck-d' || d || '.example'
FROM generate_series(1, 3) d;
VACUUM ANALYZE tenants, tenant_email_domain;
SELECT count(*) AS rows, pg_size_pretty(pg_relation_size('tenant_email_domain')) AS heap,
       pg_size_pretty(pg_indexes_size('tenant_email_domain')) AS indexes FROM tenant_email_domain;
