-- Table and index sizes.
SELECT c.relname, c.relkind, pg_size_pretty(pg_relation_size(c.oid)) AS size, pg_relation_size(c.oid) AS bytes
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname LIKE 'record%'
ORDER BY c.relkind DESC, c.relname;
SELECT pg_size_pretty(pg_total_relation_size('records')) AS records_total;
