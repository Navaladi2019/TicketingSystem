-- TICK-714 S1-S3: the RFC-015 rev 1 data layer (D2, D3, D5, D12, D19, D20),
-- as close to the RFC text as a bench allows. Runs as the superuser.
-- Throwaway: the real one is a platform migration in the PLAN.
\timing on
SET ROLE bench_owner;

-- D2 registry
CREATE TABLE record_index_fields (
    tenant_id            uuid        NOT NULL,
    id                   uuid        NOT NULL,
    entity_definition_id uuid        NOT NULL,
    field_key            text        NOT NULL,
    value_type           text        NOT NULL CHECK (value_type IN ('text', 'num', 'ts', 'bool')),
    kind                 text        NOT NULL CHECK (kind IN ('filter', 'search')),
    state                text        NOT NULL CHECK (state IN ('backfilling', 'ready', 'removing')),
    progress             smallint    NOT NULL DEFAULT 0,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT record_index_fields_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT record_index_fields_key UNIQUE (tenant_id, entity_definition_id, field_key, kind)
);

-- D3 side table
CREATE TABLE record_field_values (
    tenant_id  uuid     NOT NULL,
    field_id   uuid     NOT NULL,
    record_id  uuid     NOT NULL,
    ord        smallint NOT NULL DEFAULT 0,
    value_text text COLLATE "C",
    value_num  double precision,
    value_ts   timestamptz,
    value_bool boolean,
    CONSTRAINT record_field_values_pkey PRIMARY KEY (tenant_id, record_id, field_id, ord),
    CONSTRAINT record_field_values_one CHECK (num_nonnulls(value_text, value_num, value_ts, value_bool) = 1),
    CONSTRAINT record_field_values_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT record_field_values_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id)
);

-- D12 term table
CREATE TABLE record_search_terms (
    tenant_id uuid NOT NULL,
    field_id  uuid NOT NULL,
    term      text COLLATE "C" NOT NULL,
    record_id uuid NOT NULL,
    CONSTRAINT record_search_terms_pkey PRIMARY KEY (tenant_id, field_id, term, record_id),
    CONSTRAINT record_search_terms_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    CONSTRAINT record_search_terms_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id)
);

-- D20: forced RLS, one policy per command (bench: nava_app only).
ALTER TABLE record_index_fields ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_index_fields FORCE ROW LEVEL SECURITY;
ALTER TABLE record_field_values ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_field_values FORCE ROW LEVEL SECURITY;
ALTER TABLE record_search_terms ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_search_terms FORCE ROW LEVEL SECURITY;
CREATE POLICY rif_select ON record_index_fields FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rif_update ON record_index_fields FOR UPDATE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rif_delete ON record_index_fields FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rif_insert ON record_index_fields FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_select ON record_field_values FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_insert ON record_field_values FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rfv_delete ON record_field_values FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rst_select ON record_search_terms FOR SELECT TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rst_insert ON record_search_terms FOR INSERT TO nava_app
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY rst_delete ON record_search_terms FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- Bench: nava_app also plays nava_worker (backfill, removal), so it gets the
-- registry UPDATE/DELETE the RFC gives nava_worker.
GRANT SELECT, INSERT, UPDATE, DELETE ON record_index_fields TO nava_app;
GRANT SELECT, INSERT, DELETE ON record_field_values, record_search_terms TO nava_app;
GRANT DELETE ON records TO nava_app;
CREATE POLICY records_delete ON records FOR DELETE TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- D19 coercion: session-independent, strict JSON types.
CREATE FUNCTION record_field_coerce(v jsonb, value_type text,
    OUT value_text text, OUT value_num double precision, OUT value_ts timestamptz, OUT value_bool boolean)
LANGUAGE plpgsql STABLE PARALLEL SAFE
SET search_path = pg_catalog, public SET TimeZone = 'UTC' SET DateStyle = 'ISO, YMD' AS $f$
DECLARE s text;
BEGIN
    IF v IS NULL THEN RETURN; END IF;
    CASE value_type
    WHEN 'text' THEN
        IF jsonb_typeof(v) = 'string' THEN value_text := v #>> '{}'; END IF;
    WHEN 'num' THEN
        IF jsonb_typeof(v) = 'number' THEN
            s := v #>> '{}';
            IF pg_input_is_valid(s, 'float8') THEN value_num := s::float8; END IF;  -- overflow refused
        END IF;
    WHEN 'bool' THEN
        IF jsonb_typeof(v) = 'boolean' THEN value_bool := (v #>> '{}')::boolean; END IF;
    WHEN 'ts' THEN
        s := v #>> '{}';
        IF jsonb_typeof(v) = 'string'
           AND s ~ '^\d{4}-\d{2}-\d{2}[Tt ]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$'
           AND pg_input_is_valid(s, 'timestamptz') THEN
            value_ts := s::timestamptz;
        ELSIF jsonb_typeof(v) = 'string' AND s ~ '^\d{4}-\d{2}-\d{2}$' AND pg_input_is_valid(s, 'date') THEN
            value_ts := (s || 'T00:00:00Z')::timestamptz;
        END IF;
    END CASE;
END
$f$;

-- D12 terms: 'simple' lexemes, at most 256 distinct per field per record.
CREATE FUNCTION record_search_lexemes(v jsonb) RETURNS SETOF text
LANGUAGE sql STABLE PARALLEL SAFE SET search_path = pg_catalog, public AS $f$
SELECT lexeme FROM unnest(to_tsvector('simple', v #>> '{}')) WHERE jsonb_typeof(v) = 'string' ORDER BY lexeme LIMIT 256
$f$;

-- D5 trigger: AFTER INSERT OR UPDATE OF data; per-field diff; DELETE + INSERT
-- of changed fields only; arrays become one row per element (ord).
CREATE FUNCTION record_field_values_sync() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER
SET search_path = pg_catalog, public SET TimeZone = 'UTC' SET DateStyle = 'ISO, YMD' AS $f$
DECLARE
    changed uuid[];
BEGIN
    SELECT coalesce(array_agg(f.id), '{}') INTO changed
    FROM record_index_fields f
    WHERE f.tenant_id = NEW.tenant_id AND f.entity_definition_id = NEW.entity_definition_id
      AND f.state IN ('backfilling', 'ready')
      AND (TG_OP = 'INSERT' OR OLD.entity_definition_id IS DISTINCT FROM NEW.entity_definition_id
           OR (OLD.data -> f.field_key) IS DISTINCT FROM (NEW.data -> f.field_key));
    IF cardinality(changed) = 0 THEN
        RETURN NULL;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        DELETE FROM record_field_values WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
        DELETE FROM record_search_terms WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
    END IF;
    INSERT INTO record_field_values (tenant_id, field_id, record_id, ord, value_text, value_num, value_ts, value_bool)
    SELECT NEW.tenant_id, f.id, NEW.id, e.ord - 1, c.value_text, c.value_num, c.value_ts, c.value_bool
    FROM record_index_fields f
    CROSS JOIN LATERAL (
        SELECT x.v, x.ord FROM jsonb_array_elements(NEW.data -> f.field_key) WITH ORDINALITY x(v, ord)
        WHERE jsonb_typeof(NEW.data -> f.field_key) = 'array'
        UNION ALL
        SELECT NEW.data -> f.field_key, 1 WHERE jsonb_typeof(NEW.data -> f.field_key) <> 'array') e
    CROSS JOIN LATERAL record_field_coerce(e.v, f.value_type) c
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'filter'
      AND num_nonnulls(c.value_text, c.value_num, c.value_ts, c.value_bool) = 1;
    INSERT INTO record_search_terms (tenant_id, field_id, term, record_id)
    SELECT NEW.tenant_id, f.id, l.lexeme, NEW.id
    FROM record_index_fields f
    CROSS JOIN LATERAL record_search_lexemes(NEW.data -> f.field_key) l(lexeme)
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'search';
    RETURN NULL;
END
$f$;
CREATE FUNCTION record_field_values_purge() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER
SET search_path = pg_catalog, public AS $f$
BEGIN
    DELETE FROM record_field_values WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM record_search_terms WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    RETURN OLD;
END
$f$;
RESET ROLE;
