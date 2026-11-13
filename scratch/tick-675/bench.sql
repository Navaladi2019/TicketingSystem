-- TICK-675 (P6-10): the record_counters purge batch at volume, as a
-- nava_purge-like role under the paired policies and column grants.
-- 1,000,000 counter rows over 200 tenants: 40% default scope (NULL), 10%
-- field-only (NULL), 50% period rows with period_end spread over the last
-- 400 days to 30 days ahead, so about 46% of all rows are past the 2-day grace.
-- Run: docker run -d --name t675 -e POSTGRES_PASSWORD=x postgres:16.10
--      docker exec -i t675 psql -U postgres < scratch/tick-675/bench.sql
\timing off
SET max_parallel_workers_per_gather = 0;
CREATE ROLE p_purge;
CREATE TABLE record_counters (
    tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL,
    scope_key text NOT NULL DEFAULT '', period_end timestamptz, next bigint NOT NULL,
    PRIMARY KEY (tenant_id, entity_definition_id, scope_key));
ALTER TABLE record_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY record_counters_purge_select ON record_counters FOR SELECT TO p_purge
    USING (period_end < now() - interval '2 days');
CREATE POLICY record_counters_purge ON record_counters FOR DELETE TO p_purge
    USING (period_end < now() - interval '2 days');
GRANT SELECT (tenant_id, entity_definition_id, scope_key, period_end), DELETE ON record_counters TO p_purge;
INSERT INTO record_counters
SELECT md5('t' || (g % 200))::uuid, md5('d' || g)::uuid,
       CASE WHEN g % 10 < 4 THEN '' ELSE 'e1|' || g END,
       CASE WHEN g % 10 < 5 THEN NULL ELSE now() - ((g::bigint * 7919) % 430 - 30) * interval '1 day' END,
       2 + g % 50
FROM generate_series(1, 1000000) g
ORDER BY random();
\echo === before: no partial index
VACUUM ANALYZE record_counters;
SELECT count(*) AS rows, count(*) FILTER (WHERE period_end < now() - interval '2 days') AS expired,
       pg_size_pretty(pg_relation_size('record_counters')) AS heap FROM record_counters;
SET ROLE p_purge;
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
    WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
RESET ROLE;
\echo === after: record_counters_period_end
CREATE INDEX record_counters_period_end ON record_counters (period_end) WHERE period_end IS NOT NULL;
VACUUM ANALYZE record_counters;
SELECT pg_size_pretty(pg_relation_size('record_counters_period_end')) AS index_size;
SET ROLE p_purge;
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
    WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
\echo === steady state: everything expired already purged (the empty last batch)
RESET ROLE;
DELETE FROM record_counters WHERE period_end < now() - interval '2 days';
VACUUM ANALYZE record_counters;
SET ROLE p_purge;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
    WHERE period_end < now() - interval '2 days' LIMIT 1000);
\echo === steady state, hourly: 1,500 rows just crossed the grace (sparse), with the index
RESET ROLE;
UPDATE record_counters SET period_end = now() - interval '2 days 1 hour'
WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end IS NOT NULL ORDER BY random() LIMIT 1500);
VACUUM ANALYZE record_counters;
SET ROLE p_purge;
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
    WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
\echo === same, without the index
RESET ROLE;
DROP INDEX record_counters_period_end;
SET ROLE p_purge;
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
    WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
\echo === empty last batch, without the index
RESET ROLE;
DELETE FROM record_counters WHERE period_end < now() - interval '2 days';
VACUUM ANALYZE record_counters;
SET ROLE p_purge;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
    SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
    WHERE period_end < now() - interval '2 days' LIMIT 1000);
