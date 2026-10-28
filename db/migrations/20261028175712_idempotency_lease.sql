-- T22a (TICK-212): a lease nava_app can extend, so a takeover of an abandoned
-- in_flight idempotency row commits before the handler runs instead of holding
-- a row lock, and a connection, for the whole handler. Design and plans:
-- docs/data/idempotency-lease.md.
--
-- lease_until replaces created_at as the lease clock. The middleware writes it
-- on INSERT (now() + Lease) and a takeover moves it forward with one
-- conditional UPDATE:
--
--   UPDATE idempotency_keys SET lease_until = now() + Lease
--    WHERE tenant_id = $1 AND principal_id = $2 AND key = $3
--      AND state = 'in_flight' AND lease_until < now() AND request_hash = $4
--
-- Under READ COMMITTED a second takeover waits for the first one's commit,
-- re-checks the row and finds the lease fresh: 0 rows, 409.
--
-- No policy change: idempotency_update already scopes nava_app's UPDATE by
-- tenant in USING and WITH CHECK, so the only new right is UPDATE on
-- lease_until of nava_app's own tenant's rows (db/grants.yaml). The principal
-- is an application predicate, as for state and response_body; there is no
-- app.principal_id setting (ADR-001 A.3). nava_purge's policies still test
-- only expires_at.
--
-- The default is a stable expression, so ADD COLUMN stores it once in the
-- catalogue (attmissingval) and does not rewrite the table: the ACCESS
-- EXCLUSIVE lock lasts as long as the catalogue update, and lock_timeout keeps
-- it from queueing behind a long transaction. Rows that exist when this runs
-- get the migration time + 60 s, so no in_flight row becomes takeable earlier
-- than its 60 s lease. The 60 s matches idempotency.Lease; the middleware
-- always writes the column, so the default only serves those rows and inserts
-- that do not name it.
--
-- No index: every read and write of lease_until is a primary-key lookup.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE idempotency_keys
    ADD COLUMN lease_until timestamptz NOT NULL DEFAULT (now() + interval '60 seconds');
