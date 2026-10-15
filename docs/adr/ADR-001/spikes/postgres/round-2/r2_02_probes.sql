-- Round 2 probes. Run as superuser; each section switches role.
\set ON_ERROR_STOP 0
\timing off

\echo '=== R1 (PG-B1) relay: FIFO + fair claim uses only granted columns ==='
SET ROLE nava_relay;
BEGIN;
SELECT count(*) AS heads_claimed, count(DISTINCT tenant_id) AS tenants_served FROM (
WITH RECURSIVE tenants AS (
  (SELECT tenant_id FROM app.outbox3 WHERE dispatched_at IS NULL AND dead_at IS NULL ORDER BY tenant_id LIMIT 1)
  UNION ALL
  SELECT (SELECT o.tenant_id FROM app.outbox3 o WHERE o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.tenant_id > tenants.tenant_id ORDER BY o.tenant_id LIMIT 1)
  FROM tenants WHERE tenants.tenant_id IS NOT NULL)
SELECT c.* FROM tenants CROSS JOIN LATERAL (
  SELECT o.id, o.tenant_id FROM app.outbox3 o
  WHERE o.tenant_id = tenants.tenant_id AND o.dispatched_at IS NULL AND o.dead_at IS NULL AND o.available_at <= now()
    AND NOT EXISTS (SELECT 1 FROM app.outbox3 p WHERE p.tenant_id = o.tenant_id AND p.target_key = o.target_key
                    AND p.dispatched_at IS NULL AND p.dead_at IS NULL AND p.id < o.id)
  ORDER BY o.available_at LIMIT 2 FOR UPDATE OF o SKIP LOCKED) c) s;
UPDATE app.outbox3 SET attempts = attempts + 1, last_error = 'x', available_at = now() + interval '1 min'
WHERE tenant_id = (SELECT id FROM t WHERE n = 2) AND dispatched_at IS NULL;
ROLLBACK;
RESET ROLE;

\echo '=== R2 (PG-B1 follow-on) purge as written: FOR DELETE policy only ==='
SELECT count(*) AS purgeable_rows FROM app.outbox3 WHERE dispatched_at < now() - interval '7 days';
SET ROLE nava_purge;
BEGIN; DELETE FROM app.outbox3 WHERE dispatched_at < now() - interval '7 days'; ROLLBACK;
RESET ROLE;
\echo '--- add a matching FOR SELECT policy, same predicate ---'
SET ROLE nava_migrator;
CREATE POLICY outbox_purge_select ON app.outbox3 FOR SELECT TO nava_purge USING (dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days');
RESET ROLE;
SET ROLE nava_purge;
BEGIN; DELETE FROM app.outbox3 WHERE dispatched_at < now() - interval '7 days'; ROLLBACK;
SELECT count(*) AS purge_can_see_unexpired FROM app.outbox3 WHERE dispatched_at >= now() - interval '7 days' OR dispatched_at IS NULL;
SELECT payload FROM app.outbox3 LIMIT 1;
RESET ROLE;

\echo '=== R3 (PG-B3) PUBLIC: aclexplode misses NULL (default) ACLs ==='
CREATE FUNCTION app.created_by_bootstrap() RETURNS int LANGUAGE sql AS 'select 1';
SET ROLE nava_migrator; CREATE FUNCTION app.created_by_migrator() RETURNS int LANGUAGE sql AS 'select 1'; RESET ROLE;
SELECT p.proname, p.proacl IS NULL AS acl_is_null,
  EXISTS (SELECT 1 FROM aclexplode(p.proacl) a WHERE a.grantee = 0) AS ci3_as_written_finds_public,
  EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a WHERE a.grantee = 0) AS ci3_with_acldefault_finds_public,
  has_function_privilege('nava_relay', p.oid, 'EXECUTE') AS relay_can_execute
FROM pg_proc p WHERE p.pronamespace = 'app'::regnamespace ORDER BY 1;
SELECT 'schema app' AS obj, nspacl IS NULL AS acl_is_null FROM pg_namespace WHERE nspname = 'app'
UNION ALL SELECT 'db nava', datacl IS NULL FROM pg_database WHERE datname = 'nava';
