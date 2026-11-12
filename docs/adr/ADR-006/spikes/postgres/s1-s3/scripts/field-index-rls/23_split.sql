-- TICK-714 S1 finding: one side table per value type (D3 alternative).
-- In one shared table each value_<t> column is ~90% NULL, and the planner
-- multiplies that null fraction into every range estimate (10x low), which
-- picks a side-first plan for a 40% range + ACL (1.75 s). Per-type tables
-- with NOT NULL values plus an MCV on (tenant_id, field_id) estimate within
-- 2%. This replaces record_field_values with four tables of the same shape.
\timing on
SET max_parallel_workers_per_gather = 0;
SET maintenance_work_mem = '1GB';
DROP TABLE IF EXISTS record_field_values_num;
DROP INDEX IF EXISTS rfv_text, rfv_num, rfv_ts, rfv_bool;
DROP STATISTICS IF EXISTS rfv_text_mcv, rfv_bool_mcv, rfv_field_mcv;
DO $$
DECLARE t text; col text; typ text;
BEGIN
    FOR t, col, typ IN VALUES ('text', 'value_text', 'text COLLATE "C"'), ('num', 'value_num', 'double precision'),
                              ('ts', 'value_ts', 'timestamptz'), ('bool', 'value_bool', 'boolean') LOOP
        EXECUTE format('CREATE TABLE rfv_%s (
            tenant_id uuid NOT NULL, field_id uuid NOT NULL, record_id uuid NOT NULL, ord smallint NOT NULL DEFAULT 0,
            value %s NOT NULL,
            CONSTRAINT rfv_%s_pkey PRIMARY KEY (tenant_id, record_id, field_id, ord))', t, typ, t);
        EXECUTE format('ALTER TABLE rfv_%s OWNER TO bench_owner', t);
        EXECUTE format('INSERT INTO rfv_%s SELECT tenant_id, field_id, record_id, ord, %s FROM record_field_values
                        WHERE %s IS NOT NULL ORDER BY tenant_id, record_id, field_id', t, col, col);
        EXECUTE format('CREATE INDEX rfv_%s_value ON rfv_%s (tenant_id, field_id, value, record_id)', t, t);
        EXECUTE format('ALTER TABLE rfv_%s ADD CONSTRAINT rfv_%s_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
                        ADD CONSTRAINT rfv_%s_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id)', t, t, t);
        EXECUTE format('ALTER TABLE rfv_%s ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE rfv_%s FORCE ROW LEVEL SECURITY', t);
        EXECUTE format($p$CREATE POLICY rfv_%s_select ON rfv_%s FOR SELECT TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, t, t);
        EXECUTE format($p$CREATE POLICY rfv_%s_insert ON rfv_%s FOR INSERT TO nava_app WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, t, t);
        EXECUTE format($p$CREATE POLICY rfv_%s_delete ON rfv_%s FOR DELETE TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, t, t);
        EXECUTE format('GRANT SELECT, INSERT, DELETE ON rfv_%s TO nava_app', t);
        -- (tenant_id, field_id) for every type; (tenant_id, field_id, value) for the
        -- low-cardinality types, where equality on a value is the common filter.
        EXECUTE format('CREATE STATISTICS rfv_%s_field_mcv (mcv) ON tenant_id, field_id FROM rfv_%s', t, t);
        IF t IN ('text', 'bool') THEN
            EXECUTE format('CREATE STATISTICS rfv_%s_value_mcv (mcv) ON tenant_id, field_id, value FROM rfv_%s', t, t);
        END IF;
        EXECUTE format('ALTER STATISTICS rfv_%s_field_mcv OWNER TO bench_owner', t);
    END LOOP;
END $$;
DROP TABLE record_field_values;
ANALYZE rfv_text, rfv_num, rfv_ts, rfv_bool;

-- D5 trigger, routed by value type. Same semantics as 21_s1_schema.sql.
CREATE OR REPLACE FUNCTION record_field_values_sync() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER
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
        DELETE FROM rfv_text WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
        DELETE FROM rfv_num  WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
        DELETE FROM rfv_ts   WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
        DELETE FROM rfv_bool WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
        DELETE FROM record_search_terms WHERE tenant_id = NEW.tenant_id AND record_id = NEW.id AND field_id = ANY (changed);
    END IF;
    INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value)
    SELECT NEW.tenant_id, f.id, NEW.id, e.ord - 1, c.value_text
    FROM record_index_fields f
    CROSS JOIN LATERAL (
        SELECT x.v, x.ord FROM jsonb_array_elements(NEW.data -> f.field_key) WITH ORDINALITY x(v, ord)
        WHERE jsonb_typeof(NEW.data -> f.field_key) = 'array'
        UNION ALL
        SELECT NEW.data -> f.field_key, 1 WHERE jsonb_typeof(NEW.data -> f.field_key) <> 'array') e
    CROSS JOIN LATERAL record_field_coerce(e.v, f.value_type) c
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'filter' AND f.value_type = 'text'
      AND c.value_text IS NOT NULL;
    INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value)
    SELECT NEW.tenant_id, f.id, NEW.id, e.ord - 1, c.value_num
    FROM record_index_fields f
    CROSS JOIN LATERAL (
        SELECT x.v, x.ord FROM jsonb_array_elements(NEW.data -> f.field_key) WITH ORDINALITY x(v, ord)
        WHERE jsonb_typeof(NEW.data -> f.field_key) = 'array'
        UNION ALL
        SELECT NEW.data -> f.field_key, 1 WHERE jsonb_typeof(NEW.data -> f.field_key) <> 'array') e
    CROSS JOIN LATERAL record_field_coerce(e.v, f.value_type) c
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'filter' AND f.value_type = 'num'
      AND c.value_num IS NOT NULL;
    INSERT INTO rfv_ts (tenant_id, field_id, record_id, ord, value)
    SELECT NEW.tenant_id, f.id, NEW.id, e.ord - 1, c.value_ts
    FROM record_index_fields f
    CROSS JOIN LATERAL (
        SELECT x.v, x.ord FROM jsonb_array_elements(NEW.data -> f.field_key) WITH ORDINALITY x(v, ord)
        WHERE jsonb_typeof(NEW.data -> f.field_key) = 'array'
        UNION ALL
        SELECT NEW.data -> f.field_key, 1 WHERE jsonb_typeof(NEW.data -> f.field_key) <> 'array') e
    CROSS JOIN LATERAL record_field_coerce(e.v, f.value_type) c
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'filter' AND f.value_type = 'ts'
      AND c.value_ts IS NOT NULL;
    INSERT INTO rfv_bool (tenant_id, field_id, record_id, ord, value)
    SELECT NEW.tenant_id, f.id, NEW.id, e.ord - 1, c.value_bool
    FROM record_index_fields f
    CROSS JOIN LATERAL (
        SELECT x.v, x.ord FROM jsonb_array_elements(NEW.data -> f.field_key) WITH ORDINALITY x(v, ord)
        WHERE jsonb_typeof(NEW.data -> f.field_key) = 'array'
        UNION ALL
        SELECT NEW.data -> f.field_key, 1 WHERE jsonb_typeof(NEW.data -> f.field_key) <> 'array') e
    CROSS JOIN LATERAL record_field_coerce(e.v, f.value_type) c
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'filter' AND f.value_type = 'bool'
      AND c.value_bool IS NOT NULL;
    INSERT INTO record_search_terms (tenant_id, field_id, term, record_id)
    SELECT NEW.tenant_id, f.id, l.lexeme, NEW.id
    FROM record_index_fields f
    CROSS JOIN LATERAL record_search_lexemes(NEW.data -> f.field_key) l(lexeme)
    WHERE f.tenant_id = NEW.tenant_id AND f.id = ANY (changed) AND f.kind = 'search';
    RETURN NULL;
END
$f$;
CREATE OR REPLACE FUNCTION record_field_values_purge() RETURNS trigger LANGUAGE plpgsql SECURITY INVOKER
SET search_path = pg_catalog, public AS $f$
BEGIN
    DELETE FROM rfv_text WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM rfv_num  WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM rfv_ts   WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM rfv_bool WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    DELETE FROM record_search_terms WHERE tenant_id = OLD.tenant_id AND record_id = OLD.id;
    RETURN OLD;
END
$f$;
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size, pg_relation_size(c.oid) AS bytes
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i') AND c.relname LIKE 'rfv%' ORDER BY 1;
