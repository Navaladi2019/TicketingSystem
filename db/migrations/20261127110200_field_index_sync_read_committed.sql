-- P7-05c (TICK-979, ADR-006-E1 E1-5): the barrier's precondition fails
-- closed. record_field_values_sync() now raises, as its first statement,
-- when the writing transaction is not at READ COMMITTED.
--
-- Why: ADR-006 D9's barrier waits on every xid in progress at the flip, up
-- to a fence xid. That is enough only if a writer whose xid is above the
-- fence also read the registry after the flip committed. At READ COMMITTED
-- the trigger's registry read takes a fresh snapshot, so it does. At
-- REPEATABLE READ or SERIALIZABLE the transaction's snapshot can predate the
-- flip while its xid comes after the fence, and the record gets no side row
-- (E1-2; TA spike T2). READ UNCOMMITTED runs as READ COMMITTED in
-- PostgreSQL, but it is refused too: the rule is the setting, not the
-- behaviour.
--
-- SQLSTATE 25000 (invalid_transaction_state, class 25). The message names
-- ADR-006-E1, so a refused write is recognisable in logs (the revisit
-- trigger of AR-TICK-745-01). The check is unconditional: it runs before the
-- registry read, so a write to a type with no flagged field is refused too,
-- because "no registry rows" read from a stale snapshot is the same hole.
--
-- An UPDATE that does not name data does not fire the trigger, so it is not
-- checked. It reads no registry state either.
--
-- Everything after the check is the function of 20261115130000, unchanged:
-- the same SET list, SECURITY INVOKER and volatility. CREATE OR REPLACE
-- keeps both triggers bound to the function and takes no lock on records.
-- Cost: one current_setting() call per fired row; TICK-747's write-latency
-- run measures it. Down restores the previous body.

-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_field_values_sync() RETURNS trigger
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
    -- ADR-006-E1 E1-5: the first statement. See the header.
    IF current_setting('transaction_isolation') <> 'read committed' THEN
        RAISE EXCEPTION 'records writes must run at READ COMMITTED, not % (ADR-006-E1 E1-5)',
            current_setting('transaction_isolation')
            USING ERRCODE = 'invalid_transaction_state',
                  HINT = 'The field-index barrier (ADR-006 D9) is sound only for READ COMMITTED writers.';
    END IF;

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

-- +goose Down

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION record_field_values_sync() RETURNS trigger
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
