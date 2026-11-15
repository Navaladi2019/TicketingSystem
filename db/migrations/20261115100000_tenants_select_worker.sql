-- TICK-868 (deferred from TICK-674, P6-09; ADR-005 §3.6, A7): a create of a
-- type linked to a dated number format reads tenants.time_zone inside the
-- counter upsert (records/store scopeZoneSQL:
--   (SELECT t.time_zone FROM tenants t WHERE t.id = $1)).
-- nava_worker had no grant and no policy on tenants, so a worker-side create
-- on a dated format got no zone: the upsert wrote no row and the create
-- failed.
--
-- tenants_tenant gains nava_worker, under its existing canonical predicate on
-- the tenants key (id): the worker sees only its transaction's tenant's row,
-- as nava_app does, and with no tenant or an empty one, no row. The grant
-- (db/grants.yaml) is SELECT (id, time_zone) only, the columns the subquery
-- and the policy read: never state, onboarding_request_hash or created_at.
-- No INSERT, UPDATE or DELETE.
--
-- ALTER POLICY ... TO is catalog-only: no scan, no rewrite, no index; a brief
-- ACCESS EXCLUSIVE lock on tenants that lock_timeout bounds. tenants_pkey
-- (id) backs both the subquery and the tenant predicate (one-row index probe).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER POLICY tenants_tenant ON tenants TO nava_app, nava_worker;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER POLICY tenants_tenant ON tenants TO nava_app;
