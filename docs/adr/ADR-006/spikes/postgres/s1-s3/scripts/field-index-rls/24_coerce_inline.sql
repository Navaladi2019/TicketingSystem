-- TICK-714 S1 write-cost variant: the D19 rules unchanged, but the coerce
-- function is a plain SQL expression with NO function-level SET (so the
-- planner can inline it). The session settings are pinned once, by the
-- trigger function's own SET list (and by the backfill transaction's
-- SET LOCAL), instead of on every per-value call.
CREATE OR REPLACE FUNCTION record_field_coerce_inl(v jsonb, value_type text,
    OUT value_text text, OUT value_num double precision, OUT value_ts timestamptz, OUT value_bool boolean)
LANGUAGE sql STABLE PARALLEL SAFE AS $f$
SELECT
    CASE WHEN value_type = 'text' AND jsonb_typeof(v) = 'string' THEN v #>> '{}' END,
    CASE WHEN value_type = 'num' AND jsonb_typeof(v) = 'number' AND pg_input_is_valid(v #>> '{}', 'float8')
         THEN (v #>> '{}')::float8 END,
    CASE WHEN value_type = 'ts' AND jsonb_typeof(v) = 'string'
              AND (v #>> '{}') ~ '^\d{4}-\d{2}-\d{2}[Tt ]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$'
              AND pg_input_is_valid(v #>> '{}', 'timestamptz') THEN (v #>> '{}')::timestamptz
         WHEN value_type = 'ts' AND jsonb_typeof(v) = 'string' AND (v #>> '{}') ~ '^\d{4}-\d{2}-\d{2}$'
              AND pg_input_is_valid(v #>> '{}', 'date') THEN ((v #>> '{}') || 'T00:00:00Z')::timestamptz END,
    CASE WHEN value_type = 'bool' AND jsonb_typeof(v) = 'boolean' THEN (v #>> '{}')::boolean END
$f$;
-- Point the trigger at it (text substitution of the function name).
DO $$
DECLARE src text;
BEGIN
    SELECT pg_get_functiondef('record_field_values_sync'::regproc) INTO src;
    src := replace(src, 'record_field_coerce(', 'record_field_coerce_inl(');
    EXECUTE src;
END $$;
