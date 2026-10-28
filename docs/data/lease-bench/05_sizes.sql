-- Storage: what lease_until adds to a row once it is written, from copies of
-- the table with and without the column. (run.sh shows the heap right after
-- 00006: same filenode, same size, so no rewrite.)
\pset pager off
CREATE TEMP TABLE without_lease AS
SELECT tenant_id, principal_id, key, request_hash, state, response_status, response_body, created_at, expires_at
  FROM idempotency_keys;
CREATE TEMP TABLE with_lease AS
SELECT tenant_id, principal_id, key, request_hash, state, response_status, response_body, created_at, expires_at, lease_until
  FROM idempotency_keys;
SELECT pg_relation_size('without_lease') AS without_bytes, pg_relation_size('with_lease') AS with_bytes,
       (pg_relation_size('with_lease') - pg_relation_size('without_lease')) / 500000.0 AS bytes_per_row;
SELECT version();
