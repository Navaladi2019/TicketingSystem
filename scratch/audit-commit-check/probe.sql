SET plan_cache_mode = force_generic_plan;
SELECT relname, relpages, reltuples FROM pg_class WHERE relname IN ('audit_log_p511_young');
PREPARE p(uuid, timestamptz, xid8, text, uuid) AS SELECT 1 FROM audit_log WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3 AND subject_type = $4 AND subject_id = $5 LIMIT 1;
\echo '--- 60 rows over 20 tenants, analysed: generic plan'
EXPLAIN EXECUTE p('0190b000-0000-7000-8000-000000000001', now(), pg_current_xact_id(), 'record', gen_random_uuid());
SET enable_seqscan = off;
DEALLOCATE p;
PREPARE p(uuid, timestamptz, xid8, text, uuid) AS SELECT 1 FROM audit_log WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3 AND subject_type = $4 AND subject_id = $5 LIMIT 1;
\echo '--- same, enable_seqscan = off (the index path it rejected)'
EXPLAIN EXECUTE p('0190b000-0000-7000-8000-000000000001', now(), pg_current_xact_id(), 'record', gen_random_uuid());
RESET enable_seqscan;
\echo '--- 60 rows of ONE tenant, analysed'
BEGIN;
SET session_replication_role = replica;
DELETE FROM audit_log_p511_young;
INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
 SELECT '0190b000-0000-7000-8000-000000000001', gen_random_uuid(), date_trunc('month', now()) + (now() - date_trunc('month', now())) * (g::float8/61), (2000000+g)::text::xid8, 'record','update','record', gen_random_uuid(), 'workflow','workflow' FROM generate_series(1,60) g;
SET session_replication_role = DEFAULT;
ANALYZE audit_log;
DEALLOCATE p;
PREPARE p(uuid, timestamptz, xid8, text, uuid) AS SELECT 1 FROM audit_log WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3 AND subject_type = $4 AND subject_id = $5 LIMIT 1;
EXPLAIN EXECUTE p('0190b000-0000-7000-8000-000000000001', now(), pg_current_xact_id(), 'record', gen_random_uuid());
\echo '--- partition grows to 4,060 rows inside a transaction (an import), stats still say 60'
SET session_replication_role = replica;
INSERT INTO audit_log (tenant_id, id, occurred_at, txid, domain, action, subject_type, subject_id, actor_system, source)
 SELECT '0190c000-0000-7000-8000-000000000099', gen_random_uuid(), now(), pg_current_xact_id(), 'record','create','record', gen_random_uuid(), 'workflow','workflow' FROM generate_series(1,4000) g;
SET session_replication_role = DEFAULT;
DEALLOCATE p;
PREPARE p(uuid, timestamptz, xid8, text, uuid) AS SELECT 1 FROM audit_log WHERE tenant_id = $1 AND occurred_at = $2 AND txid = $3 AND subject_type = $4 AND subject_id = $5 LIMIT 1;
EXPLAIN EXECUTE p('0190c000-0000-7000-8000-000000000099', now(), pg_current_xact_id(), 'record', gen_random_uuid());
ROLLBACK;
SELECT version();
