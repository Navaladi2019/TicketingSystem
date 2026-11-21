-- TICK-1059: cost of acl2_problem before (TICK-940) and after the under-set
-- cap, per call, on three shapes. Run inside a migrated nava database:
--   psql -v ON_ERROR_STOP=1 -f bench.sql   (after the 20261121161036 Up)
-- The shapes: A, 128 member_of leaves, no under (today's documents);
-- B, 64 policies each with 2 under leaves of distinct ids (128 leaves, 4 U
-- each, accepted); C, one condition of 120 under leaves (8 x 15) of distinct ids
-- reffed by one policy (refused: under_sets_exceeded).
\timing off
CREATE TEMP TABLE shapes(name text, doc jsonb);
CREATE FUNCTION pg_temp.u(n int) RETURNS text LANGUAGE sql AS
    $f$ SELECT '"0b0b0b0b-0000-4000-8000-' || lpad(n::text, 12, '0') || '"' $f$;
INSERT INTO shapes SELECT 'A no under, 128 leaves', ('{"format": 2, "conditions": [], "policies": [' ||
    string_agg(format('{"name": "p%s", "ops": ["read"], "when": {"any": [{"member_of": {"groups": [%s]}}, {"member_of": {"groups": [%s]}}]}}',
        i, pg_temp.u(2*i), pg_temp.u(2*i+1)), ', ') || ']}')::jsonb FROM generate_series(1, 64) i;
INSERT INTO shapes SELECT 'B 64 policies x 2 under', ('{"format": 2, "conditions": [], "policies": [' ||
    string_agg(format('{"name": "p%s", "ops": ["read"], "when": {"any": [{"under": {"of": [%s]}}, {"under": {"of": [%s]}}]}}',
        i, pg_temp.u(2*i), pg_temp.u(2*i+1)), ', ') || ']}')::jsonb FROM generate_series(1, 64) i;
INSERT INTO shapes SELECT 'C 120 under in a ref', ('{"format": 2, "conditions": [{"name": "c", "expr": {"any": [' ||
    string_agg(g, ', ') || ']}}], "policies": [{"name": "p", "ops": ["read"], "when": {"ref": "c"}}]}')::jsonb
FROM (SELECT '{"any": [' || string_agg(format('{"under": {"of": [%s]}}', pg_temp.u(16*gi+k)), ', ') || ']}' AS g
      FROM generate_series(0, 7) gi, generate_series(0, 14) k GROUP BY gi) s;

DO $$
DECLARE r record; t0 timestamptz; n int := 400; p jsonb;
BEGIN
    FOR r IN SELECT * FROM shapes ORDER BY name LOOP
        p := acl2_problem(r.doc);
        t0 := clock_timestamp();
        FOR i IN 1 .. n LOOP PERFORM acl2_problem(r.doc); END LOOP;
        RAISE NOTICE '% | % bytes | % us/call | %', r.name, octet_length(r.doc::text),
            round(extract(epoch FROM clock_timestamp() - t0) * 1e6 / n), coalesce(p ->> 'code', 'valid');
    END LOOP;
END $$;
