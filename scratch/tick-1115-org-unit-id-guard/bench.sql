-- TICK-1115: cost of entity_definition_versions_org_unit_id_reserved against
-- the plain `schema -> 'properties' -> 'org_unit_id'`. Escapes are built with
-- chr(92), so no tool can decode them.
\set ON_ERROR_STOP on
SELECT version();

CREATE FUNCTION reserved(s json) RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT (replace(replace(replace(replace(s::text,
        '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json
        -> 'properties' -> 'org_unit_id') IS NOT NULL $$;

-- 10,000 rows each. plain: 40 properties, no escapes. dense: strings made of
-- escapes; every fifth row is close to the 128 KiB schema cap.
CREATE TABLE plain (s json);
INSERT INTO plain
SELECT ('{"type":"object","additionalProperties":false,"properties":{' || string_agg(
    '"p' || j || '":{"type":"string","title":"Field ' || j || '","maxLength":200}', ',') || '}}')::json
FROM generate_series(1, 10000) i, generate_series(1, 40) j GROUP BY i;

CREATE TABLE dense (s json);
INSERT INTO dense
SELECT ('{"description":"' || repeat('ab' || chr(92) || 'n' || chr(92) || 'u0041' || chr(92) || '"' || chr(92) || chr(92),
    CASE WHEN i % 5 = 0 THEN 8700 ELSE 330 END) || '","properties":{"a":{}}}')::json
FROM generate_series(1, 10000) i;

SELECT 'plain' AS t, count(*), avg(octet_length(s::text))::int AS avg_bytes, max(octet_length(s::text)) AS max_bytes,
       sum((strpos(s::text, chr(92) || 'u0041') > 0)::int) AS rows_with_escape FROM plain
UNION ALL
SELECT 'dense', count(*), avg(octet_length(s::text))::int, max(octet_length(s::text)),
       sum((strpos(s::text, chr(92) || 'u0041') > 0)::int) FROM dense;

\timing on
\echo plain, plain arrow x3
ALTER TABLE plain ADD CONSTRAINT c CHECK ((s -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE plain DROP CONSTRAINT c;
ALTER TABLE plain ADD CONSTRAINT c CHECK ((s -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE plain DROP CONSTRAINT c;
ALTER TABLE plain ADD CONSTRAINT c CHECK ((s -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE plain DROP CONSTRAINT c;
\echo plain, chosen x3
ALTER TABLE plain ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE plain DROP CONSTRAINT c;
ALTER TABLE plain ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE plain DROP CONSTRAINT c;
ALTER TABLE plain ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE plain DROP CONSTRAINT c;
\echo dense, plain arrow x3
ALTER TABLE dense ADD CONSTRAINT c CHECK ((s -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE dense DROP CONSTRAINT c;
ALTER TABLE dense ADD CONSTRAINT c CHECK ((s -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE dense DROP CONSTRAINT c;
ALTER TABLE dense ADD CONSTRAINT c CHECK ((s -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE dense DROP CONSTRAINT c;
\echo dense, chosen x3
ALTER TABLE dense ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE dense DROP CONSTRAINT c;
ALTER TABLE dense ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE dense DROP CONSTRAINT c;
ALTER TABLE dense ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL); ALTER TABLE dense DROP CONSTRAINT c;

\echo insert 10,000 plain rows, no CHECK x3, then with the CHECK x3
CREATE TABLE ins (s json);
INSERT INTO ins SELECT s FROM plain; TRUNCATE ins;
INSERT INTO ins SELECT s FROM plain; TRUNCATE ins;
INSERT INTO ins SELECT s FROM plain; TRUNCATE ins;
ALTER TABLE ins ADD CONSTRAINT c CHECK ((replace(replace(replace(replace(s::text, '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json -> 'properties' -> 'org_unit_id') IS NULL);
INSERT INTO ins SELECT s FROM plain; TRUNCATE ins;
INSERT INTO ins SELECT s FROM plain; TRUNCATE ins;
INSERT INTO ins SELECT s FROM plain; TRUNCATE ins;
\timing off

-- The same expression on schemas built from bytes: the answer it must give.
\pset format aligned
WITH f(want, name, s) AS (VALUES
    (true,  'plain spelling',          '{"properties":{"org_unit_id":{}}}'),
    (true,  'NUL escape beside it',    '{"d":"' || chr(92) || 'u0000","properties":{"org_unit_id":{}}}'),
    (true,  'o escaped',               '{"properties":{"' || chr(92) || 'u006frg_unit_id":{}}}'),
    (true,  'p escaped',               '{"' || chr(92) || 'u0070roperties":{"org_unit_id":{}}}'),
    (false, 'NUL escape',              '{"d":"' || chr(92) || 'u0000","properties":{}}'),
    (false, 'lone high',               '{"d":"' || chr(92) || 'ud800","properties":{}}'),
    (false, 'lone high, upper case',   '{"d":"' || chr(92) || 'uD83D","properties":{}}'),
    (false, 'lone low',                '{"d":"' || chr(92) || 'udc00","properties":{}}'),
    (false, 'escaped pair',            '{"d":"' || chr(92) || 'ud83d' || chr(92) || 'ude00","properties":{}}'),
    (false, 'escaped backslash u0000', '{"d":"' || chr(92) || chr(92) || 'u0000","properties":{}}'),
    (false, 'key plus NUL escape',     '{"properties":{"org_unit_id' || chr(92) || 'u0000":{}}}'),
    (false, 'nested',                  '{"properties":{"a":{"properties":{"org_unit_id":{}}}}}'))
SELECT name, want, reserved(s::json) AS got, want = reserved(s::json) AS ok FROM f;
