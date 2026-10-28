-- T22a (TICK-212) lease benchmark seed, run as pgadmin with 00006 undone
-- (02 re-applies it over these rows):
--   idempotency_keys 500,000 rows over 25 h, the purge-bench shape: 2,000
--                    principals; every 10th row in a small tenant, the rest in
--                    one big tenant (t1). 1% in_flight with no response, all
--                    abandoned (created 2 to 10 minutes ago), the rest
--                    completed with a ~550-byte body.
-- Ages, placement and ids are md5-derived: runs repeat.
\timing on
CREATE FUNCTION pg_temp.u(text) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$ SELECT md5($1)::uuid $$;
CREATE FUNCTION pg_temp.r(text) RETURNS float8 LANGUAGE sql IMMUTABLE AS
    $$ SELECT ('x' || substr(md5($1), 1, 8))::bit(32)::bigint / 4294967296.0 $$;
CREATE FUNCTION pg_temp.t(int) RETURNS uuid LANGUAGE sql IMMUTABLE AS
    $$ SELECT pg_temp.u('t' || CASE WHEN $1 % 10 = 0 THEN 2 ELSE 1 END) $$;

INSERT INTO tenants (id, state) VALUES (pg_temp.u('t1'), 'active'), (pg_temp.u('t2'), 'active');

-- in_flight: i % 100 = 7 (1%). 02 (after 00006) gives every row
-- lease_until = created_at + 60 s, as the middleware would have.
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, state, response_status,
                              response_body, created_at, expires_at)
SELECT pg_temp.t(i), pg_temp.u('p' || (i % 2000)), 'k' || i, decode(md5('h' || i), 'hex'),
       CASE WHEN i % 100 = 7 THEN 'in_flight' ELSE 'completed' END,
       CASE WHEN i % 100 = 7 THEN NULL ELSE 201 END,
       CASE WHEN i % 100 = 7 THEN NULL ELSE convert_to(repeat('{"id":"' || i || '"}', 25), 'UTF8') END,
       c, c + interval '24 hours'
FROM (SELECT i, CASE WHEN i % 100 = 7 THEN now() - interval '2 minutes' - pg_temp.r('ic' || i) * interval '8 minutes'
                     ELSE now() - pg_temp.r('ic' || i) * interval '25 hours' END AS c
      FROM generate_series(1, 500000) i ORDER BY md5('ip' || i)) s;

VACUUM (ANALYZE) idempotency_keys;

\pset pager off
SELECT state, count(*), count(*) FILTER (WHERE created_at < now() - interval '60 seconds') AS past_60s
FROM idempotency_keys GROUP BY state ORDER BY state;
SELECT pg_relation_size('idempotency_keys') AS heap_bytes, pg_relation_filenode('idempotency_keys') AS filenode;
