-- :tenants tenants, :rows rows, override :nd, occurred_at spread over 30 days at microsecond
-- grain (nearly every value distinct, as real entries are), 1 to 2 entries
-- per txid. The R10 assert is off for the load only.
\set ON_ERROR_STOP on
TRUNCATE audit_log;
TRUNCATE tenants CASCADE;
INSERT INTO tenants SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid FROM generate_series(1, :tenants) g;
ALTER TABLE audit_log DISABLE TRIGGER audit_log_assert_insert;
ALTER TABLE audit_log DROP CONSTRAINT IF EXISTS audit_log_tenant_fkey;
INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, version_after,
                       actor_system, source)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(1 + (g % :tenants)), 12, '0'))::uuid,
       gen_random_uuid(),
       timestamptz '2026-10-01' + (g / 2) * interval '1 microsecond' * (2592000000000 / :rows),
       (1000 + g / 2)::text::xid8,
       'record', 'update', 'record', gen_random_uuid(), 1 + g % 7, 'workflow', 'workflow'
FROM generate_series(1, :rows) g;
ALTER TABLE audit_log ENABLE TRIGGER audit_log_assert_insert;
-- The override under test (:nd), on the parent and the partition.
ALTER TABLE audit_log ALTER COLUMN occurred_at SET (n_distinct = :nd);
ALTER TABLE audit_log_default ALTER COLUMN occurred_at SET (n_distinct = :nd);
ANALYZE audit_log;
SELECT count(*) AS rows, count(DISTINCT tenant_id) AS tenants FROM audit_log;
SELECT attname, n_distinct, histogram_bounds IS NOT NULL AS has_histogram FROM pg_stats WHERE tablename = 'audit_log_default' AND attname IN ('tenant_id', 'occurred_at', 'txid', 'subject_id') ORDER BY attname;
-- A time-window read (export, reader) uses the histogram, not n_distinct:
-- its row estimate must not move with the override. One tenant, 3 of 30 days.
EXPLAIN SELECT count(*) FROM audit_log WHERE tenant_id = '00000000-0000-4000-8000-000000000001' AND occurred_at >= '2026-10-10' AND occurred_at < '2026-10-13';
SELECT count(*) AS actual_rows_in_window FROM audit_log WHERE tenant_id = '00000000-0000-4000-8000-000000000001' AND occurred_at >= '2026-10-10' AND occurred_at < '2026-10-13';
