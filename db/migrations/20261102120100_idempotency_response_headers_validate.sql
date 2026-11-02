-- TICK-412: validates the previous migration's response_headers CHECK in its
-- own transaction. VALIDATE CONSTRAINT scans idempotency_keys under SHARE
-- UPDATE EXCLUSIVE, so the middleware's reads and writes go on while it runs.
-- Every existing row has response_headers NULL, so nothing is rejected.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE idempotency_keys VALIDATE CONSTRAINT idempotency_keys_response_headers_check;
