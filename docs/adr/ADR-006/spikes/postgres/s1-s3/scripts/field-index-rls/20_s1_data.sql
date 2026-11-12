-- TICK-714 S1-S3 (RFC-015 rev 1, d471c036 §10). Runs as the superuser after
-- `bench/records-list/run.sh setup custom` (800k records, one 400k tenant).
-- Throwaway: not a migration.
-- 1. Drops the pre-rev-1 spike tables.
-- 2. Puts the large tenant's 400k records into ONE entity type (the
--    requirement is 100k+ per type; D2 field ids are per type).
-- 3. Adds the S1 fields at fixed selectivities (hash of the record id, so
--    they reproduce and are independent of updated_at and of the ACL) and
--    two free-text fields for S2.
\timing on
SET max_parallel_workers_per_gather = 0;
SET maintenance_work_mem = '1GB';
DROP TRIGGER IF EXISTS records_rfv_sync ON records;
DROP FUNCTION IF EXISTS rfv_sync();
DROP TABLE IF EXISTS record_field_values, record_field_values2, record_field_values_numeric, entity_field_index, bench_field, bench_actor_f;
DROP FUNCTION IF EXISTS rfv_coerce(jsonb, text);

-- Words w1..w20000; word k is picked log-uniformly, so a few words are
-- common and most are rare (Zipf-like). (A vocabulary-table lookup per word
-- cost 5 ms per call: 2 hours for 800k rows. This is 15 us.)
DROP TABLE IF EXISTS bench_vocab, bench_vocab_arr;
DROP FUNCTION IF EXISTS bench_words2(text, int);
CREATE OR REPLACE FUNCTION bench_words(seed text, n int) RETURNS text LANGUAGE sql IMMUTABLE AS $f$
SELECT string_agg('w' || (1 + floor(exp((abs(hashtext(seed || ':' || i)) % 1000000) / 1000000.0 * ln(20000)))::int), ' ' ORDER BY i)
FROM generate_series(1, n) i
$f$;

CREATE OR REPLACE FUNCTION bench_h(id uuid, salt text) RETURNS int LANGUAGE sql IMMUTABLE AS
$f$ SELECT abs(hashtext(id::text || salt)) % 100000 $f$;

UPDATE records r SET
    entity_definition_id = CASE WHEN r.tenant_id = md5('tenant0')::uuid THEN md5('ed0/0')::uuid ELSE r.entity_definition_id END,
    data = r.data || jsonb_build_object(
        -- text eq: 40% / 2% / 0.25% / 0.01%, the rest spread over 50 values
        'tier', CASE WHEN h.t < 40000 THEN 'a40' WHEN h.t < 42000 THEN 'b2' WHEN h.t < 42250 THEN 'c025'
                     WHEN h.t < 42260 THEN 'd001' ELSE 'z' || (h.t % 50) END,
        -- text in (two values that together hit 40% / 2% / 0.25% / 0.01%)
        'zone', CASE WHEN h.z < 20000 THEN 'p40a' WHEN h.z < 40000 THEN 'p40b' WHEN h.z < 41000 THEN 'p2a'
                     WHEN h.z < 42000 THEN 'p2b' WHEN h.z < 42125 THEN 'p025a' WHEN h.z < 42250 THEN 'p025b'
                     WHEN h.z < 42255 THEN 'p001a' WHEN h.z < 42260 THEN 'p001b' ELSE 'q' || (h.z % 50) END,
        -- num: uniform [0, 10000); lt 1 / 25 / 200 / 4000 = 0.01 / 0.25 / 2 / 40%
        'score', h.s / 10.0,
        -- ts: 2025-01-01Z + h minutes; lt +10 / +250 / +2000 / +40000 minutes
        'opened', to_char(timestamptz '2025-01-01 00:00:00+00' + h.o * interval '1 minute', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
        'b001', h.b1 < 10, 'b025', h.b2 < 250, 'b2', h.b3 < 2000, 'b40', h.b4 < 40000,
        -- S2 free text
        'summary', bench_words(r.id::text || 's', 12),
        'notes', bench_words(r.id::text || 'n', 40))
FROM (SELECT id AS hid, tenant_id AS htid, bench_h(id, 'tier') AS t, bench_h(id, 'zone') AS z, bench_h(id, 'score') AS s,
             bench_h(id, 'opened') AS o, bench_h(id, 'b001') AS b1, bench_h(id, 'b025') AS b2,
             bench_h(id, 'b2') AS b3, bench_h(id, 'b40') AS b4
      FROM records) h
WHERE h.htid = r.tenant_id AND h.hid = r.id;
VACUUM FULL records;
VACUUM (ANALYZE) records;
SELECT count(*) AS large_tenant,
       count(*) FILTER (WHERE data ->> 'tier' = 'b2') AS tier_b2,
       count(*) FILTER (WHERE (data ->> 'score')::float8 < 25) AS score_lt_25,
       count(*) FILTER (WHERE (data ->> 'b001')::boolean) AS b001,
       round(avg(pg_column_size(data))) AS avg_data_bytes
FROM records WHERE tenant_id = md5('tenant0')::uuid;
SELECT pg_size_pretty(pg_relation_size('records')) AS records_heap;
