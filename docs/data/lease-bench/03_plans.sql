-- The statements of a takeover as nava_app, RLS on, in the big tenant t1, on
-- one abandoned key, rolled back so each run starts from the seed. Then T22's
-- FOR UPDATE NOWAIT on the same row for comparison.
\pset pager off
SELECT md5('t1')::uuid AS t1 \gset

BEGIN;
SELECT set_config('app.tenant_id', :'t1', true) \g /dev/null
SELECT principal_id AS p, key AS k, encode(request_hash, 'hex') AS h FROM idempotency_keys
 WHERE tenant_id = :'t1' AND state = 'in_flight' AND lease_until < now() ORDER BY key LIMIT 1 \gset
-- the read (selectRow)
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT request_hash, state, response_status, response_body, lease_until < now()
  FROM idempotency_keys
 WHERE tenant_id = :'t1' AND principal_id = :'p' AND key = :'k';
-- the takeover (takeOver)
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
UPDATE idempotency_keys
   SET lease_until = now() + 60 * interval '1 second'
 WHERE tenant_id = :'t1' AND principal_id = :'p' AND key = :'k'
   AND state = 'in_flight' AND lease_until < now() AND request_hash = decode(:'h', 'hex');
-- a second takeover of the same row finds the lease fresh: 0 rows
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
UPDATE idempotency_keys
   SET lease_until = now() + 60 * interval '1 second'
 WHERE tenant_id = :'t1' AND principal_id = :'p' AND key = :'k'
   AND state = 'in_flight' AND lease_until < now() AND request_hash = decode(:'h', 'hex');
ROLLBACK;

BEGIN;
SELECT set_config('app.tenant_id', :'t1', true) \g /dev/null
-- T22 (before): the row lock its retake held for the whole handler
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT state, response_status, response_body FROM idempotency_keys
 WHERE tenant_id = :'t1' AND principal_id = :'p' AND key = :'k'
   FOR UPDATE NOWAIT;
ROLLBACK;
