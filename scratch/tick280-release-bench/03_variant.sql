-- Variant: a (tenant_id) index, to price what the release would gain.
CREATE INDEX tenant_email_domain_tenant_idx ON tenant_email_domain (tenant_id);
VACUUM ANALYZE tenant_email_domain;
SELECT pg_size_pretty(pg_relation_size('tenant_email_domain_tenant_idx')) AS tenant_idx,
       pg_size_pretty(pg_relation_size('tenant_email_domain_pkey')) AS pkey;
