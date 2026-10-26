\pset pager off
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size, pg_relation_size(c.oid) AS bytes
FROM pg_class c
WHERE c.relname IN ('outbox', 'workflow_ref', 'idempotency_keys')
   OR c.oid IN (SELECT indexrelid FROM pg_index WHERE indrelid IN
       ('outbox'::regclass, 'workflow_ref'::regclass, 'idempotency_keys'::regclass))
ORDER BY c.relname;
SELECT version();
SHOW shared_buffers;
