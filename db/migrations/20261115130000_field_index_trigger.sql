-- P7-02 (TICK-742): record_field_coerce and the records triggers that keep
-- the field-index side tables equal to a recompute from records.data
-- (ADR-006 D1, D4, D5, R5). Design note: docs/data/field-index-trigger.md.
--
-- record_field_coerce(v, value_type) is the one coercion. The trigger here
-- and P7-05's backfill both call it; nothing re-implements it in Go. It
-- returns one row per value that coerces, with ord (the position in a
-- top-level array, 0 for a scalar) and the value in the column of its type,
-- and no row for anything else:
--   text  a JSON string of at most 2,000 bytes (a longer one would not fit
--         a B-tree entry of rfv_text_value and would fail the write)
--   num   a JSON number that is a finite double precision (1e400 and
--         1e-400 are out of range; NaN can only be a string, which is refused)
--   bool  a JSON boolean
--   ts    an RFC 3339 date-time with an explicit offset (Z or +hh:mm), or a
--         YYYY-MM-DD date, stored as 00:00 UTC. The write pipeline asserts
--         format (internal/metadata), so only a format: date field holds a
--         bare date, and no field holds a date-time without an offset.
--   JSON null, an absent key, a nested array, an object, and array
--   elements past ord 32,767 (smallint) give no row.
-- STABLE, not IMMUTABLE: text to timestamptz is stable. The SET list pins
-- every setting the casts read, so the same value gives the same rows in any
-- session (CI-9): TimeZone for the offset-free date, DateStyle for the
-- parser, search_path for name resolution.
--
-- The triggers are plpgsql, SECURITY INVOKER (ADR-001 I3): they run as the
-- writer, under its RLS, with the same SET list.
--   AFTER INSERT OR UPDATE OF data, FOR EACH ROW: for each registry row of
--   the record's type in backfilling or ready whose data -> field_key changed
--   (IS DISTINCT FROM; on INSERT, every one), DELETE the record's rows for the
--   field, then INSERT the coerced values. It reads only the registry and
--   NEW/OLD, never a side table (D1).
--   BEFORE DELETE, FOR EACH ROW: delete the record's side rows, so the NO
--   ACTION record FKs pass at the end of the statement.
--
-- A value that does not coerce writes no row and never fails the write (D5).
-- The trigger counts them (non-null values minus rows written) and adds the
-- count to the transaction-local setting nava.field_index_uncoerced, a JSON
-- object {"<entity_definition_id>": n}. For a type that flags a field,
-- internal/records/pipeline reads it in the commit's round trip
-- (tenancy.ReadSettingAtCommit) and adds each count
-- to nava_field_index_uncoerced_total{entity_definition}. No RAISE, no
-- table write: a rolled-back savepoint or transaction takes its counts back
-- with it. A setting value the trigger did not write is discarded, never
-- cast blindly, so it cannot fail a write either.
--
-- A writer the registry policies do not cover sees no registry row and
-- copies nothing. nava_app and nava_worker are covered
-- (TestP701EveryRecordWriterReadsTheRegistry). The owner, nava_migrator, has
-- no registry policy under FORCE RLS: a data migration that writes records
-- as the migrator indexes nothing, and must run the P7-05 rebuild afterwards.
-- A DELETE of a record with side rows as the migrator fails on the record FK
-- (its purge sees no side rows), loudly, never leaving an orphan.
--
-- Grants: EXECUTE on record_field_coerce to nava_app and nava_worker
-- (db/grants.yaml), who call it through the trigger and the backfill. A
-- trigger function needs no EXECUTE grant to fire. PUBLIC has none (00001).
--
-- Locks: CREATE TRIGGER takes SHARE ROW EXCLUSIVE on records for
-- milliseconds (catalogue only, no scan); lock_timeout makes the migration
-- fail and be re-run instead of queueing record writes behind a long
-- transaction. The registry is empty until P7-04 publishes a flag, so the
-- triggers copy nothing until then.

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- +goose StatementBegin
CREATE FUNCTION record_field_coerce(v jsonb, value_type text)
    RETURNS TABLE (ord smallint, value_text text, value_num double precision, value_ts timestamptz, value_bool boolean)
    LANGUAGE sql STABLE PARALLEL SAFE
    SET search_path = pg_catalog, public
    SET "TimeZone" = 'UTC'
    SET "DateStyle" = 'ISO, YMD'
AS $f$
    WITH e AS (
        SELECT a.n - 1 AS ord, a.x FROM jsonb_array_elements(v) WITH ORDINALITY a(x, n)
        WHERE jsonb_typeof(v) = 'array'
        UNION ALL
        SELECT 0, v WHERE jsonb_typeof(v) <> 'array'
    ), s AS (
        SELECT e.ord, e.x, jsonb_typeof(e.x) AS t, e.x #>> '{}' AS str FROM e WHERE e.ord <= 32767
    ), c AS (
        SELECT s.ord::smallint AS ord,
            CASE WHEN value_type = 'text' AND s.t = 'string' AND octet_length(s.str) <= 2000
                 THEN s.str END AS value_text,
            CASE WHEN value_type = 'num' AND s.t = 'number' AND pg_input_is_valid(s.str, 'float8')
                 THEN s.str::double precision END AS value_num,
            CASE WHEN value_type <> 'ts' OR s.t <> 'string' THEN NULL
                 WHEN s.str ~ '^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):[0-5][0-9]:([0-5][0-9]|60)(\.[0-9]+)?([Zz]|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$'
                      AND pg_input_is_valid(s.str, 'timestamptz')
                 THEN s.str::timestamptz
                 WHEN s.str ~ '^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$'
                      AND pg_input_is_valid(s.str, 'date')
                 THEN s.str::date::timestamp AT TIME ZONE 'UTC' END AS value_ts,
            CASE WHEN value_type = 'bool' AND s.t = 'boolean'
                 THEN s.x::boolean END AS value_bool
        FROM s
    )
    SELECT c.ord, c.value_text, c.value_num, c.value_ts, c.value_bool FROM c
    WHERE CASE value_type
        WHEN 'text' THEN c.value_text IS NOT NULL
        WHEN 'num'  THEN c.value_num IS NOT NULL AND c.value_num NOT IN ('Infinity', '-Infinity')
        WHEN 'ts'   THEN c.value_ts IS NOT NULL AND isfinite(c.value_ts)
        WHEN 'bool' THEN c.value_bool IS NOT NULL
        ELSE false END
$f$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION record_field_values_sync() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER
    SET search_path = pg_catalog, public
    SET "TimeZone" = 'UTC'
    SET "DateStyle" = 'ISO, YMD'
AS $f$
DECLARE
    f       record;
    v       jsonb;
    values_ bigint;
    written bigint;
    missed  bigint := 0;
    cur     text;
    acc     jsonb := '{}';
    def_key text;
    prev    bigint := 0;
BEGIN
    FOR f IN
        SELECT r.id, r.field_key, r.value_type FROM record_index_fields r
        WHERE r.tenant_id = NEW.tenant_id AND r.entity_definition_id = NEW.entity_definition_id
          AND r.state IN ('backfilling', 'ready')
    LOOP
        v := NEW.data -> f.field_key;
        IF TG_OP = 'UPDATE' THEN
            IF (OLD.data -> f.field_key) IS NOT DISTINCT FROM v THEN
                CONTINUE;
            END IF;
            CASE f.value_type
            WHEN 'text' THEN
                DELETE FROM rfv_text WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = f.id;
            WHEN 'num' THEN
                DELETE FROM rfv_num WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = f.id;
            WHEN 'ts' THEN
                DELETE FROM rfv_ts WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = f.id;
            WHEN 'bool' THEN
                DELETE FROM rfv_bool WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = f.id;
            END CASE;
        END IF;
        IF v IS NULL OR jsonb_typeof(v) = 'null' THEN
            CONTINUE;
        END IF;
        CASE f.value_type
        WHEN 'text' THEN
            INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value)
            SELECT NEW.tenant_id, f.id, NEW.id, c.ord, c.value_text FROM record_field_coerce(v, 'text') c;
        WHEN 'num' THEN
            INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value)
            SELECT NEW.tenant_id, f.id, NEW.id, c.ord, c.value_num FROM record_field_coerce(v, 'num') c;
        WHEN 'ts' THEN
            INSERT INTO rfv_ts (tenant_id, field_id, record_id, ord, value)
            SELECT NEW.tenant_id, f.id, NEW.id, c.ord, c.value_ts FROM record_field_coerce(v, 'ts') c;
        WHEN 'bool' THEN
            INSERT INTO rfv_bool (tenant_id, field_id, record_id, ord, value)
            SELECT NEW.tenant_id, f.id, NEW.id, c.ord, c.value_bool FROM record_field_coerce(v, 'bool') c;
        END CASE;
        GET DIAGNOSTICS written = ROW_COUNT;
        IF jsonb_typeof(v) = 'array' THEN
            SELECT count(*) INTO values_ FROM jsonb_array_elements(v) x WHERE jsonb_typeof(x) <> 'null';
        ELSE
            values_ := 1;
        END IF;
        missed := missed + values_ - written;
    END LOOP;

    IF missed > 0 THEN
        -- Only a JSON object of whole numbers this trigger wrote is kept;
        -- anything else in the setting is discarded, never cast unchecked.
        cur := current_setting('nava.field_index_uncoerced', true);
        IF cur IS NOT NULL AND pg_input_is_valid(cur, 'jsonb') THEN
            IF jsonb_typeof(cur::jsonb) = 'object' THEN
                acc := cur::jsonb;
            END IF;
        END IF;
        def_key := NEW.entity_definition_id::text;
        IF jsonb_typeof(acc -> def_key) = 'number' THEN
            IF pg_input_is_valid(acc ->> def_key, 'bigint') THEN
                prev := (acc ->> def_key)::bigint;
            END IF;
        END IF;
        PERFORM set_config('nava.field_index_uncoerced',
            (acc || jsonb_build_object(def_key, prev + missed))::text, true);
    END IF;
    RETURN NULL;
END
$f$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION record_field_values_purge() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER
    SET search_path = pg_catalog, public
    SET "TimeZone" = 'UTC'
    SET "DateStyle" = 'ISO, YMD'
AS $f$
BEGIN
    DELETE FROM rfv_text WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM rfv_num  WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM rfv_ts   WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM rfv_bool WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    RETURN OLD;
END
$f$;
-- +goose StatementEnd

CREATE TRIGGER records_field_values_sync
    AFTER INSERT OR UPDATE OF data ON records
    FOR EACH ROW EXECUTE FUNCTION record_field_values_sync();
CREATE TRIGGER records_field_values_purge
    BEFORE DELETE ON records
    FOR EACH ROW EXECUTE FUNCTION record_field_values_purge();

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TRIGGER records_field_values_purge ON records;
DROP TRIGGER records_field_values_sync ON records;
DROP FUNCTION record_field_values_purge();
DROP FUNCTION record_field_values_sync();
DROP FUNCTION record_field_coerce(jsonb, text);
