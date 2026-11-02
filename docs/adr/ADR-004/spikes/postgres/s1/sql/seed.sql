-- Spike S1 synthetic data. 50 tenants. Tenant 1 holds 1,000,000 sealed audit
-- entries (item 5's "one tenant among many"); tenants 2-50 hold 20,000 each.
-- 20,000 records per tenant (1,000,000 records). Ids are md5-derived so the
-- pgbench scripts can address rows without a lookup.
\set ON_ERROR_STOP on
\timing on
SET session_replication_role = replica; -- load only: skips the R10/R11 triggers
INSERT INTO records (tenant_id, id, entity_definition_id, number, version, data, created_at, updated_at)
SELECT md5('tenant-' || t)::uuid, md5('rec-' || t || '-' || r)::uuid, md5('def-' || t)::uuid, r, 1,
       jsonb_build_object('title', 'Ticket ' || r, 'status', r % 5, 'priority', r % 3,
                          'assignee', md5('user-' || r)::uuid, 'description', repeat('lorem ipsum ', 20)),
       now() - interval '90 days', now() - interval '1 day'
  FROM generate_series(1, 50) t, generate_series(1, 20000) r;

CREATE TEMP TABLE n_per_tenant AS
SELECT t, CASE WHEN t = 1 THEN 1000000 ELSE 20000 END AS n FROM generate_series(1, 50) t;

INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id,
                       entity_definition_id, version_before, version_after, actor_principal_id, source,
                       request_id, content_sha256, diff, diff_salt, diff_sha256,
                       seq, prev_hash, entry_hash, sealed_at)
SELECT md5('tenant-' || p.t)::uuid,
       gen_random_uuid(),
       now() - interval '90 days' + (g * (interval '89 days' / p.n)),
       (p.t::bigint * 10000000 + g)::text::xid8,
       'record', 'update', 'record',
       md5('rec-' || p.t || '-' || (g % 20000 + 1))::uuid,
       md5('def-' || p.t)::uuid, g / 20000, g / 20000 + 1,
       md5('user-' || (g % 997))::uuid, 'user',
       'req-' || g,
       sha256(int8send(g)),
       ('[{"op":"replace","path":"/status","value":' || (g % 5) || '},{"op":"replace","path":"/description","value":"' || repeat('x', 120) || '"}]')::json,
       decode(md5(g::text), 'hex'),
       sha256(int8send(-g)),
       g,
       CASE WHEN g = 1 THEN '\x0000000000000000000000000000000000000000000000000000000000000000'::bytea ELSE sha256(int8send(g - 1)) END,
       sha256(int8send(g)),
       now() - interval '90 days' + (g * (interval '89 days' / p.n)) + interval '10 seconds'
  FROM n_per_tenant p, LATERAL generate_series(1, p.n) g;

INSERT INTO audit_chain_head (tenant_id, seq, entry_hash)
SELECT md5('tenant-' || t)::uuid, n, sha256(int8send(n)) FROM n_per_tenant;
RESET session_replication_role;
VACUUM (ANALYZE) records;
VACUUM (ANALYZE) audit_log;
VACUUM (ANALYZE) audit_chain_head;
CHECKPOINT;
SELECT 'records' AS t, count(*) FROM records UNION ALL SELECT 'audit_log', count(*) FROM audit_log;
SELECT relname, pg_size_pretty(pg_total_relation_size(oid)) AS total, pg_size_pretty(pg_relation_size(oid)) AS heap
  FROM pg_class WHERE relname IN ('records','audit_log_default') ORDER BY 1;
SELECT indexrelid::regclass AS index, pg_size_pretty(pg_relation_size(indexrelid)) AS size
  FROM pg_index WHERE indrelid IN ('audit_log_default'::regclass, 'records'::regclass) ORDER BY 1;
