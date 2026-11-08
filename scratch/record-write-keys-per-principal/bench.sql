-- TICK-626: replay lookup on record_write_keys, tenant-wide primary key vs
-- the per-principal unique index. Run on a throwaway postgres:16.10:
--   docker run -d --name rwk -e POSTGRES_PASSWORD=x postgres:16.10
--   docker exec -i rwk psql -U postgres -f - < bench.sql
-- The table matches record_write_keys's columns; foreign keys and RLS are
-- left out (they do not change the plan of a lookup by key).
\timing off
DROP TABLE IF EXISTS rwk;
CREATE TABLE rwk (
    tenant_id          uuid    NOT NULL,
    idempotency_key    text    NOT NULL,
    definition_id      uuid    NOT NULL,
    action             text    NOT NULL,
    record_id          uuid    NOT NULL,
    number             text    NOT NULL,
    version            bigint  NOT NULL,
    schema_version     integer NOT NULL,
    audit_entry_id     uuid    NOT NULL,
    actor_principal_id uuid
);
-- 1,000 tenants x 2,000 keys = 2,000,000 rows. 20 principals per tenant,
-- one in ten keys written by a system writer (NULL). Keys look like the
-- pipeline's: wf:<uuid>:step-n.
INSERT INTO rwk
SELECT ('00000000-0000-7000-8000-' || lpad(to_hex(t), 12, '0'))::uuid,
       'wf:' || md5(t::text || ':' || k) || ':step-' || (k % 7),
       gen_random_uuid(), CASE WHEN k % 3 = 0 THEN 'update' ELSE 'create' END,
       gen_random_uuid(), 'TCK' || lpad(k::text, 7, '0'), 1 + k % 5, 1, gen_random_uuid(),
       CASE WHEN k % 10 = 0 THEN NULL
            ELSE ('00000000-0000-7000-9000-' || lpad(to_hex(t * 100 + k % 20), 12, '0'))::uuid END
FROM generate_series(1, 1000) t, generate_series(1, 2000) k;

\echo '== before: PRIMARY KEY (tenant_id, idempotency_key)'
ALTER TABLE rwk ADD CONSTRAINT rwk_pkey PRIMARY KEY (tenant_id, idempotency_key);
VACUUM ANALYZE rwk;
SELECT pg_size_pretty(pg_relation_size('rwk')) AS heap, pg_size_pretty(pg_relation_size('rwk_pkey')) AS pkey;
-- main's WriteKeyByKey: no actor.
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT definition_id, action, record_id, number, version, schema_version, audit_entry_id
FROM rwk WHERE tenant_id = '00000000-0000-7000-8000-0000000001f4'
  AND idempotency_key = 'wf:' || md5('500:1234') || ':step-2';

\echo '== after: UNIQUE (tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT'
\timing on
CREATE UNIQUE INDEX CONCURRENTLY rwk_actor_key ON rwk (tenant_id, idempotency_key, actor_principal_id) NULLS NOT DISTINCT;
ALTER TABLE rwk DROP CONSTRAINT rwk_pkey;
\timing off
VACUUM ANALYZE rwk;
SELECT pg_size_pretty(pg_relation_size('rwk_actor_key')) AS actor_key;
-- WriteKeyByKey with a member: hit (k = 1234: principal t*100 + 14).
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT definition_id, action, record_id, number, version, schema_version, audit_entry_id
FROM rwk WHERE tenant_id = '00000000-0000-7000-8000-0000000001f4'
  AND idempotency_key = 'wf:' || md5('500:1234') || ':step-2'
  AND actor_principal_id IS NOT DISTINCT FROM '00000000-0000-7000-9000-00000000c35e'::uuid;
-- Another member under the same key: miss, the AR-TICK-480-01 path.
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT definition_id, action, record_id, number, version, schema_version, audit_entry_id
FROM rwk WHERE tenant_id = '00000000-0000-7000-8000-0000000001f4'
  AND idempotency_key = 'wf:' || md5('500:1234') || ':step-2'
  AND actor_principal_id IS NOT DISTINCT FROM '00000000-0000-7000-9000-00000000c350'::uuid;
-- A system writer (NULL actor): k = 1230 is a NULL row.
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT definition_id, action, record_id, number, version, schema_version, audit_entry_id
FROM rwk WHERE tenant_id = '00000000-0000-7000-8000-0000000001f4'
  AND idempotency_key = 'wf:' || md5('500:1230') || ':step-5'
  AND actor_principal_id IS NOT DISTINCT FROM NULL::uuid;

-- Write cost: 20,000 keyed inserts against each index shape.
\echo '== insert cost, 20,000 rows'
\timing on
INSERT INTO rwk SELECT tenant_id, idempotency_key || ':n', definition_id, action, record_id, number, version,
    schema_version, audit_entry_id, actor_principal_id FROM rwk WHERE tenant_id <= '00000000-0000-7000-8000-00000000000a';
\timing off
DROP INDEX rwk_actor_key;
DELETE FROM rwk WHERE idempotency_key LIKE '%:n';
VACUUM rwk;
ALTER TABLE rwk ADD CONSTRAINT rwk_pkey PRIMARY KEY (tenant_id, idempotency_key);
\timing on
INSERT INTO rwk SELECT tenant_id, idempotency_key || ':n', definition_id, action, record_id, number, version,
    schema_version, audit_entry_id, actor_principal_id FROM rwk WHERE tenant_id <= '00000000-0000-7000-8000-00000000000a';
\timing off
