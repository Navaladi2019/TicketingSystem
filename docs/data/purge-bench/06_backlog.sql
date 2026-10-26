-- A purge outage: half of every table past its window, committed, as the
-- bootstrap superuser. 03_plans.sql then shows whether the index plans still
-- touch about one batch of rows per batch when the expired set is large.
\timing on
UPDATE idempotency_keys SET expires_at = now() - interval '1 hour'
WHERE expires_at >= now() AND substr(md5(key), 1, 1) < '8';
UPDATE outbox SET dispatched_at = now() - interval '8 days'
WHERE dispatched_at >= now() - interval '7 days' AND substr(md5(dedupe_key), 1, 1) < '8';
UPDATE workflow_ref SET closed_at = now() - interval '31 days'
WHERE closed_at >= now() - interval '30 days' AND substr(md5(outbox_id::text), 1, 1) < '8';
VACUUM (ANALYZE) outbox, workflow_ref, idempotency_keys;
\pset pager off
SELECT 'outbox' AS t, count(*) AS rows,
       count(*) FILTER (WHERE dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days') AS expired
FROM outbox
UNION ALL
SELECT 'workflow_ref', count(*), count(*) FILTER (WHERE closed_at < now() - interval '30 days') FROM workflow_ref
UNION ALL
SELECT 'idempotency_keys', count(*), count(*) FILTER (WHERE expires_at < now()) FROM idempotency_keys;
