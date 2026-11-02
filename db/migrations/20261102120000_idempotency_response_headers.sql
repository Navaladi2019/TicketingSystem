-- TICK-412: a replayed idempotent response restores an allow-listed set of
-- response headers (ETag, Location), not only Content-Type. TICK-357's
-- POST /api/v1/records/{entity} answers 201 with ETag: W/"<version>" and
-- Location; a client that retries after a timeout gets the replay and needs
-- that ETag for its next If-Match (TICK-359). Design and figures:
-- docs/data/idempotency-response-headers.md.
--
-- A column of its own, not a prefix inside response_body: the body stays the
-- handler's bytes, so a replay remains byte-identical (CI-8) without parsing
-- an envelope. jsonb, not one column per header: the allow-list lives in the
-- middleware (idempotency.ReplayHeaders), so adding a header later is a code
-- change, not DDL. The value is a JSON object of header name to one string,
-- for example {"ETag": "W/\"1\"", "Location": "/api/v1/records/incident/…"};
-- NULL when the response had none of the allow-listed headers, and on every
-- row written before this migration (their replays keep the old behaviour).
--
-- The CHECK keeps the top-level type an object: the middleware scans it into
-- map[string]string and pgx refuses anything else, so a stray row would turn
-- its replay into a 500. Members are the middleware's job.
--
-- No default and nullable, so ADD COLUMN touches only the catalogue (no
-- rewrite). The CHECK is added NOT VALID here, under the same short ACCESS
-- EXCLUSIVE lock, and validated by the next migration under SHARE UPDATE
-- EXCLUSIVE, which blocks neither the middleware's reads nor its writes.
-- New rows are checked from this migration on.
--
-- No policy change: idempotency_update already scopes nava_app's UPDATE by
-- tenant in USING and WITH CHECK, and the purge policies test only
-- expires_at. The new right is UPDATE on response_headers (db/grants.yaml).
-- No index: every read and write is a primary-key lookup.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE idempotency_keys
    ADD COLUMN response_headers jsonb,
    ADD CONSTRAINT idempotency_keys_response_headers_check
        CHECK (response_headers IS NULL OR jsonb_typeof(response_headers) = 'object') NOT VALID;
