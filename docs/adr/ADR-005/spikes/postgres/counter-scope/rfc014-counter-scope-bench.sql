-- Minimal copy of today's record_counters (20261101190000_records_pipeline.sql:139)
-- plus the roles and policies that matter. Volume: 2,000 tenants x 10 types = 20,000
-- empty-scope rows (today's shape; prod holds a few thousand).
CREATE ROLE nava_app; CREATE ROLE nava_purge;
CREATE TABLE entity_definitions (tenant_id uuid NOT NULL, id uuid NOT NULL, PRIMARY KEY (tenant_id, id));
CREATE TABLE record_counters (
    tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL, next bigint NOT NULL,
    CONSTRAINT record_counters_pkey PRIMARY KEY (tenant_id, entity_definition_id),
    CONSTRAINT record_counters_entity_definition_fkey FOREIGN KEY (tenant_id, entity_definition_id)
        REFERENCES entity_definitions (tenant_id, id),
    CONSTRAINT record_counters_next_check CHECK (next >= 2));
ALTER TABLE record_counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_counters FORCE ROW LEVEL SECURITY;
CREATE POLICY rc_select ON record_counters FOR SELECT TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rc_insert ON record_counters FOR INSERT TO nava_app WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rc_update ON record_counters FOR UPDATE TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid) WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT, INSERT, UPDATE ON record_counters TO nava_app;
CREATE TABLE t (n int, id uuid);
INSERT INTO t SELECT g, md5('tenant'||g)::uuid FROM generate_series(1,2000) g;
INSERT INTO entity_definitions SELECT t.id, md5(t.id::text||d)::uuid FROM t, generate_series(1,10) d;
INSERT INTO record_counters SELECT tenant_id, id, 4001 FROM entity_definitions;
VACUUM ANALYZE;
SELECT version(), count(*) AS counter_rows, pg_size_pretty(pg_total_relation_size('record_counters')) FROM record_counters;
\timing on
\echo '== M1 expand (one transaction): column with constant default = catalog only, no rewrite'
SELECT pg_relation_filenode('record_counters') AS filenode_before \gset
BEGIN;
SET LOCAL lock_timeout = '2s';
ALTER TABLE record_counters ADD COLUMN scope_key text NOT NULL DEFAULT '';
ALTER TABLE record_counters ADD CONSTRAINT record_counters_scope_key_check CHECK (length(scope_key) <= 128) NOT VALID;
COMMIT;
ALTER TABLE record_counters VALIDATE CONSTRAINT record_counters_scope_key_check;
SELECT pg_relation_filenode('record_counters') = :filenode_before AS no_rewrite;
\echo '== M2 (NO TRANSACTION): new unique index, built online'
DROP INDEX CONCURRENTLY IF EXISTS record_counters_scope_uq;
CREATE UNIQUE INDEX CONCURRENTLY record_counters_scope_uq ON record_counters (tenant_id, entity_definition_id, scope_key);
\echo '== interim: OLD code (2-col ON CONFLICT) and NEW code (3-col) both work'
SET ROLE nava_app;
BEGIN; SELECT set_config('app.tenant_id', md5('tenant1')::uuid::text, true);
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 2)
 ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 AS old_code;
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, '', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1 AS new_code;
\echo '-- interim: a non-empty scope still hits the old 2-col PK, so templates must wait for M3'
SAVEPOINT s;
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:202611', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK TO s; COMMIT;
RESET ROLE;
\echo '== M3 contract (one transaction): swap the PK onto the new index'
BEGIN;
SET LOCAL lock_timeout = '2s';
ALTER TABLE record_counters DROP CONSTRAINT record_counters_pkey,
  ADD CONSTRAINT record_counters_pkey PRIMARY KEY USING INDEX record_counters_scope_uq;
COMMIT;
\d record_counters
\echo '== after M3: OLD code fails on every create'
SET ROLE nava_app;
BEGIN; SELECT set_config('app.tenant_id', md5('tenant1')::uuid::text, true);
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 2)
 ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK; RESET ROLE;
-- Upsert plan at today's volume (20,000 rows), as nava_app under FORCE RLS.
SET ROLE nava_app;
BEGIN; SELECT set_config('app.tenant_id', md5('tenant7')::uuid::text, true) \gset
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant7')::uuid, md5(md5('tenant7')::uuid::text||3)::uuid, '', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK; RESET ROLE;
-- Scope rows at the RFC's growth bound, skewed. period_end = the instant the period
-- closes (NULL for field-only scopes); it is the proposed purge column.
ALTER TABLE record_counters ADD COLUMN period_end timestamptz;
-- big tenant (tenant1): 2 types x 1,000 branches x 366 days ({field:branch} + {dd}) = 732,000
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next, period_end)
SELECT md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||d)::uuid,
       format('e1|d:%s|f:B%s', to_char(day,'YYYYMMDD'), b), 2 + (random()*50)::int, day + interval '1 day'
FROM generate_series(1,2) d, generate_series(1,1000) b, generate_series(timestamptz '2025-11-09', '2026-11-09', '1 day') day;
-- 500 tenants x 10 types x 36 months (monthly reset, 3 years) = 180,000
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next, period_end)
SELECT t.id, md5(t.id::text||d)::uuid, format('e1|d:%s', to_char(m,'YYYYMM')), 2 + (random()*500)::int, m + interval '1 month'
FROM t, generate_series(1,10) d, generate_series(timestamptz '2023-12-01', '2026-11-01', '1 month') m WHERE t.n BETWEEN 2 AND 501;
-- 100 tenants x 5 types x 1,000 branch codes, field-only scope (never purged) = 500,000
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next)
SELECT t.id, md5(t.id::text||d)::uuid, format('e1|f:B%s', b), 2 + (random()*500)::int
FROM t, generate_series(1,5) d, generate_series(1,1000) b WHERE t.n BETWEEN 502 AND 601;
VACUUM ANALYZE record_counters;
SELECT count(*) rows, count(period_end) period_rows, count(*) FILTER (WHERE period_end < now() - interval '2 days') purgeable,
  pg_size_pretty(pg_relation_size('record_counters')) heap, pg_size_pretty(pg_relation_size('record_counters_pkey')) pkey
FROM record_counters;
-- Upsert at 1,432,000 rows, as nava_app under FORCE RLS. Each run twice; the second is steady state.
SET ROLE nava_app;
\echo '== existing scope row, big tenant (tenant1, daily x branch)'
BEGIN; SELECT set_config('app.tenant_id', md5('tenant1')::uuid::text, true) \gset
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:20261109|f:B517', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:20261109|f:B517', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
\echo '== first create in a new scope (insert path; FK check on entity_definitions runs)'
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:20261110|f:B517', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK;
\echo '== small tenant, empty scope (today''s default template)'
BEGIN; SELECT set_config('app.tenant_id', md5('tenant1500')::uuid::text, true) \gset
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1500')::uuid, md5(md5('tenant1500')::uuid::text||3)::uuid, '', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1500')::uuid, md5(md5('tenant1500')::uuid::text||3)::uuid, '', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK; RESET ROLE;
-- Steady state: the warm-up commits in its own transaction, so the explained upsert
-- updates a row written by another transaction (as a real create does) and the FK
-- check is skipped (key columns unchanged).
SET ROLE nava_app;
BEGIN; SELECT set_config('app.tenant_id', md5('tenant1')::uuid::text, true) \gset
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:20261109|f:B517', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
COMMIT;
\echo '== 1.43M rows, big tenant, existing daily x branch scope'
BEGIN; SELECT set_config('app.tenant_id', md5('tenant1')::uuid::text, true) \gset
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:20261109|f:B517', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK;
\echo '== cross-tenant: tenant 2 session targeting tenant1 counter row'
BEGIN; SELECT set_config('app.tenant_id', md5('tenant2')::uuid::text, true) \gset
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1')::uuid, md5(md5('tenant1')::uuid::text||1)::uuid, 'e1|d:20261109|f:B517', 2)
 ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1;
ROLLBACK; RESET ROLE;
-- Purge as nava_purge (cross-tenant, paired SELECT/DELETE policies, same as outbox).
-- Batch shape copies internal/purge/purge.go: DELETE ... WHERE pk IN (SELECT ... LIMIT $1).
CREATE POLICY rc_purge_select ON record_counters FOR SELECT TO nava_purge USING (period_end < now() - interval '2 days');
CREATE POLICY rc_purge ON record_counters FOR DELETE TO nava_purge USING (period_end < now() - interval '2 days');
GRANT SELECT, DELETE ON record_counters TO nava_purge;
SET ROLE nava_purge;
\echo '== A. predicate parsed out of scope_key (no period_end column): first batch of 1,000'
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters
  WHERE scope_key ~ '\|d:\d{8}' AND to_date(substring(scope_key from '\|d:(\d{8})'),'YYYYMMDD') < now() - interval '3 days' LIMIT 1000);
ROLLBACK;
\echo '== B. period_end, no index: first batch, then the empty last batch'
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
DELETE FROM record_counters WHERE period_end < now() - interval '2 days';
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
RESET ROLE;
CREATE INDEX record_counters_purge ON record_counters (period_end) WHERE period_end IS NOT NULL;
ANALYZE record_counters;
SELECT pg_size_pretty(pg_relation_size('record_counters_purge')) purge_index_size;
SET ROLE nava_purge;
\echo '== C. period_end + partial index: first batch, then the empty last batch'
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
DELETE FROM record_counters WHERE period_end < now() - interval '2 days';
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
RESET ROLE;
-- Steady state: purge has run before, so only the newest closed periods are due.
-- Delete the backlog, vacuum, then age 2,000 day-scope rows (one day x 2 types x 1,000 branches).
DELETE FROM record_counters WHERE period_end < now() - interval '2 days';
VACUUM ANALYZE record_counters;
UPDATE record_counters SET period_end = now() - interval '3 days' WHERE scope_key LIKE 'e1|d:20261109|%';
VACUUM ANALYZE record_counters;
SELECT count(*) rows, count(period_end) period_rows, count(*) FILTER (WHERE period_end < now() - interval '2 days') due FROM record_counters;
SET ROLE nava_purge;
\echo '== C steady: partial index, batch of 1,000'
BEGIN;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
\echo '== C steady: the empty last batch'
DELETE FROM record_counters WHERE period_end < now() - interval '2 days';
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
RESET ROLE;
\echo '== B steady: same, index dropped'
BEGIN; DROP INDEX record_counters_purge; SET LOCAL ROLE nava_purge;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
DELETE FROM record_counters WHERE (tenant_id, entity_definition_id, scope_key) IN (
  SELECT tenant_id, entity_definition_id, scope_key FROM record_counters WHERE period_end < now() - interval '2 days' LIMIT 1000);
ROLLBACK;
\echo '== HOT check: 20,000 single-row upserts on one counter row, with the partial index present'
SELECT pg_stat_reset_single_table_counters('record_counters'::regclass);
DO $$ BEGIN
  PERFORM set_config('app.tenant_id', md5('tenant1500')::uuid::text, false);
  FOR i IN 1..20000 LOOP
    INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES (md5('tenant1500')::uuid, md5(md5('tenant1500')::uuid::text||3)::uuid, '', 2)
    ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1;
  END LOOP; END $$;
SELECT pg_sleep(1);
SELECT n_tup_upd, n_tup_hot_upd FROM pg_stat_user_tables WHERE relname = 'record_counters';
