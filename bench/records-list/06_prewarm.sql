-- Warm cache: every relation of records and record_shares into shared_buffers.
SELECT c.relname, pg_prewarm(c.oid) AS blocks
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname LIKE 'record%' AND c.relkind IN ('r', 'i')
ORDER BY c.relname;
