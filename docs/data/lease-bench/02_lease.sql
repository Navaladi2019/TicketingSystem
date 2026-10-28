-- After 00006: every row's lease as the middleware would have written it
-- (created_at + 60 s), instead of the migration-time default.
\timing on
UPDATE idempotency_keys SET lease_until = created_at + interval '60 seconds';
VACUUM (FULL, ANALYZE) idempotency_keys;
\pset pager off
SELECT state, count(*), count(*) FILTER (WHERE lease_until < now()) AS lease_passed
FROM idempotency_keys GROUP BY state ORDER BY state;
