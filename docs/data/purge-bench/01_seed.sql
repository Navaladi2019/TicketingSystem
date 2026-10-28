-- TICK-146 benchmark seed, run as the bootstrap superuser (RLS does not apply)
-- after db-roles and every migration. Two tenants (t1 holds 90% of the rows,
-- t2 10%), shaped as the tables look an hour after the last purge run with
-- the ADR-001 A.3 windows: only the last hour's rows are past their window.
--   outbox           1,000,000 rows over 7 d 1 h; 1% pending, 0.3% dead
--                    (dead_at over 31 d), the rest dispatched
--   workflow_ref       500,000 rows over 30 d 1 h; 2% open
--   idempotency_keys   500,000 rows over 25 h; expires_at = created_at + 24 h
--                    (T22 has not fixed the TTL; 24 h is the assumption)
-- Rows go in in an order unrelated to their age: after purge and vacuum, new
-- rows reuse freed space anywhere in the heap, so expired rows are not all
-- at its start. Ages, placement and ids are md5-derived: runs repeat.
\timing on
CREATE FUNCTION pg_temp.u(text) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$ SELECT md5($1)::uuid $$;
-- r(s): a number in [0, 1) derived from s.
CREATE FUNCTION pg_temp.r(text) RETURNS float8 LANGUAGE sql IMMUTABLE AS
    $$ SELECT ('x' || substr(md5($1), 1, 8))::bit(32)::bigint / 4294967296.0 $$;
CREATE FUNCTION pg_temp.t(int) RETURNS uuid LANGUAGE sql IMMUTABLE AS
    $$ SELECT pg_temp.u('t' || CASE WHEN $1 % 10 = 0 THEN 2 ELSE 1 END) $$;

INSERT INTO tenants (id, state) VALUES (pg_temp.u('t1'), 'active'), (pg_temp.u('t2'), 'active');

-- age: 0 (newest) .. 1 (oldest) of each table's span.
INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload,
                    created_at, available_at, dispatched_at, dead_at, attempts)
SELECT pg_temp.u('o' || i), pg_temp.t(i), 'record.changed', 'd' || i, 'rec:' || (i % 50000), i,
       -- References only (00008 CHECK, CI-7).
       jsonb_build_object('record_id', pg_temp.u('r' || (i % 50000)),
                          'watchers', jsonb_build_array(pg_temp.u('w' || i), pg_temp.u('x' || i))),
       c, c,
       CASE WHEN i % 100 = 0 OR i % 333 = 0 THEN NULL ELSE c + interval '2 seconds' END,
       CASE WHEN i % 333 = 0 THEN now() - (i % 31) * interval '1 day' - interval '1 minute' END,
       CASE WHEN i % 333 = 0 THEN 20 ELSE 1 END
FROM (SELECT i, now() - pg_temp.r('oc' || i) * interval '169 hours' AS c
      FROM generate_series(1, 1000000) i ORDER BY md5('op' || i)) s;

INSERT INTO records (tenant_id, id)
SELECT pg_temp.t(i), pg_temp.u('wr' || i) FROM generate_series(1, 500000) i;

INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, created_at, closed_at)
SELECT pg_temp.t(i), pg_temp.u('wo' || i), pg_temp.u('wr' || i), 'approval', c,
       CASE WHEN i % 50 = 0 THEN NULL ELSE c + interval '5 minutes' END
FROM (SELECT i, now() - interval '5 minutes' - pg_temp.r('wc' || i) * interval '721 hours' AS c
      FROM generate_series(1, 500000) i ORDER BY md5('wp' || i)) s;

INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, state, response_status,
                              response_body, created_at, expires_at)
SELECT pg_temp.t(i), pg_temp.u('p' || (i % 2000)), 'k' || i, decode(md5('h' || i), 'hex'),
       'completed', 201, convert_to(repeat('{"id":"' || i || '"}', 25), 'UTF8'), c, c + interval '24 hours'
FROM (SELECT i, now() - pg_temp.r('ic' || i) * interval '25 hours' AS c
      FROM generate_series(1, 500000) i ORDER BY md5('ip' || i)) s;

VACUUM (ANALYZE) outbox, records, workflow_ref, idempotency_keys;

\pset pager off
SELECT 'outbox' AS t, count(*) AS rows,
       count(*) FILTER (WHERE dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days') AS expired
FROM outbox
UNION ALL
SELECT 'workflow_ref', count(*), count(*) FILTER (WHERE closed_at < now() - interval '30 days') FROM workflow_ref
UNION ALL
SELECT 'idempotency_keys', count(*), count(*) FILTER (WHERE expires_at < now()) FROM idempotency_keys;
