-- The release as nava_onboard, each in a transaction rolled back so the
-- seed stays: the stuck provisioning tenant (3 rows freed), then an active
-- tenant's own transaction (0 rows, the guard), then the large active tenant.
\set stuck '00000000-0000-4000-8000-0000000fffff'
\set active '00000000-0000-4000-8000-000000000005'
\set big '00000000-0000-4000-8000-000000002711'
BEGIN;
SELECT set_config('app.tenant_id', :'stuck', true) IS NOT NULL AS set;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM tenant_email_domain WHERE tenant_id = :'stuck';
ROLLBACK;
BEGIN;
SELECT set_config('app.tenant_id', :'active', true) IS NOT NULL AS set;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM tenant_email_domain WHERE tenant_id = :'active';
ROLLBACK;
BEGIN;
SELECT set_config('app.tenant_id', :'big', true) IS NOT NULL AS set;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) DELETE FROM tenant_email_domain WHERE tenant_id = :'big';
ROLLBACK;
