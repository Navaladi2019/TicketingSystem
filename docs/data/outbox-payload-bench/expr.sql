-- TICK-240: the CHECK expressions alone, over 200,000 typical payloads
-- ({"record_id": uuid, "watchers": [uuid, uuid]}), so the figure is the
-- evaluation cost with no WAL or heap work. Prints one line per run:
-- label, rows passing (must be 200000), milliseconds.
CREATE TEMP TABLE src AS
SELECT jsonb_build_object('record_id', gen_random_uuid(),
                          'watchers', jsonb_build_array(gen_random_uuid(), gen_random_uuid())) AS payload
FROM generate_series(1, 200000);
ANALYZE src;

CREATE OR REPLACE FUNCTION pg_temp.timed(label text, expr text) RETURNS text LANGUAGE plpgsql AS $$
DECLARE t0 timestamptz; n bigint;
BEGIN
    t0 := clock_timestamp();
    EXECUTE 'SELECT count(*) FILTER (WHERE ' || expr || ') FROM src' INTO n;
    RETURN format('%s %s %s', label, n, round(extract(epoch FROM clock_timestamp() - t0) * 1000));
END $$;

SELECT pg_temp.timed(l, e) FROM (VALUES
  ('none', $e$true$e$),
  ('v6', $e$jsonb_typeof(payload) = 'object'
    AND NOT jsonb_path_exists(payload, 'strict $.* ? (
        !(@.type() == "string"
          && @ like_regex "^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[uU][rR][nN]:[uU][uU][iI][dD]:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[[:ascii:]][0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}[[:ascii:]]|[0-9a-fA-F]{32})$")
        && !(@.type() == "array"
             && !exists(@[*] ? (
                 !(@.type() == "string"
                   && @ like_regex "^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[uU][rR][nN]:[uU][uU][iI][dD]:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[[:ascii:]][0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}[[:ascii:]]|[0-9a-fA-F]{32})$")))))')$e$),
  ('v7', $e$CASE WHEN jsonb_typeof(payload) = 'object' THEN
        NOT jsonb_path_exists(payload, 'strict $.keyvalue() ? (
            !(@.key like_regex "^[a-z][a-z0-9_]{0,31}$")
            || !((@.value.type() == "string"
                  && @.value like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
                 || (@.value.type() == "array"
                     && !exists(@.value[*] ? (
                         !(@.type() == "string"
                           && @ like_regex "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"))))))')
    ELSE false END$e$)
) AS c(l, e), generate_series(1, 5) AS r ORDER BY r, l;
