-- TICK-412: storage and lock cost of idempotency_keys.response_headers.
-- 200k completed rows (about a day of keys for a busy deployment), the
-- 00002 shape plus lease_until; then the two migrations' statements, timed.
\timing on
CREATE TABLE idempotency_keys (
    tenant_id uuid NOT NULL, principal_id uuid NOT NULL, key text NOT NULL,
    request_hash bytea NOT NULL, state text NOT NULL DEFAULT 'in_flight',
    response_status smallint, response_body bytea,
    created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL,
    lease_until timestamptz NOT NULL DEFAULT now() + interval '60 seconds',
    PRIMARY KEY (tenant_id, principal_id, key));
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, state, response_status, response_body, expires_at)
SELECT md5((i % 500)::text)::uuid, md5((i % 5000)::text || 'p')::uuid, gen_random_uuid()::text,
       sha256(i::text::bytea), 'completed', 201,
       convert_to(format('{"id":"%s","version":1,"data":{"title":"incident %s","severity":"high"}}', gen_random_uuid(), i), 'UTF8'),
       now() + interval '1 day'
  FROM generate_series(1, 200000) i;
VACUUM ANALYZE idempotency_keys;
SELECT 'before' AS at, pg_size_pretty(pg_table_size('idempotency_keys')) AS heap, pg_table_size('idempotency_keys') AS bytes;

BEGIN;
SET LOCAL lock_timeout = '5s';
ALTER TABLE idempotency_keys ADD COLUMN response_headers jsonb,
  ADD CONSTRAINT idempotency_keys_response_headers_check
  CHECK (response_headers IS NULL OR jsonb_typeof(response_headers) = 'object') NOT VALID;
COMMIT;
BEGIN;
ALTER TABLE idempotency_keys VALIDATE CONSTRAINT idempotency_keys_response_headers_check;
COMMIT;
SELECT 'after migration (all NULL)' AS at, pg_size_pretty(pg_table_size('idempotency_keys')) AS heap, pg_table_size('idempotency_keys') AS bytes;

-- Steady state: every row written with both headers (worst case: every write is a create).
CREATE TABLE ik2 AS SELECT * FROM idempotency_keys WITH NO DATA;
INSERT INTO ik2 SELECT tenant_id, principal_id, key, request_hash, state, response_status, response_body, created_at, expires_at, lease_until,
  jsonb_build_object('ETag', 'W/"1"', 'Location', '/api/v1/records/incident/' || gen_random_uuid())
  FROM idempotency_keys;
VACUUM ANALYZE ik2;
SELECT 'every row with ETag + Location' AS at, pg_size_pretty(pg_table_size('ik2')) AS heap, pg_table_size('ik2') AS bytes;
SELECT avg(pg_column_size(response_headers))::int AS avg_value_bytes FROM ik2;
