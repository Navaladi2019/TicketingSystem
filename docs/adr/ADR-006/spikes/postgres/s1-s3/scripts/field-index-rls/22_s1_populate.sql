-- TICK-714 S1/S2: registry rows and the initial copy, set-based as the
-- superuser (the bench shortcut for "every field already ready"); then the
-- D3 indexes, the FKs (validated once) and the D5 triggers.
\timing on
SET max_parallel_workers_per_gather = 0;
SET maintenance_work_mem = '1GB';
ALTER TABLE record_field_values DROP CONSTRAINT record_field_values_record_fk, DROP CONSTRAINT record_field_values_field_fk;
ALTER TABLE record_search_terms DROP CONSTRAINT record_search_terms_record_fk, DROP CONSTRAINT record_search_terms_field_fk;

-- Ten filterable fields on every type of every tenant (the D14 per-type quota).
INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, kind, state, progress)
SELECT d.tenant_id, md5('fld' || d.tenant_id || d.entity_definition_id || f.k || 'filter')::uuid, d.entity_definition_id,
       f.k, f.t, 'filter', 'ready', 100
FROM (SELECT DISTINCT tenant_id, entity_definition_id FROM records) d,
     (VALUES ('tier', 'text'), ('zone', 'text'), ('site', 'text'), ('category', 'text'), ('score', 'num'),
             ('opened', 'ts'), ('b001', 'bool'), ('b025', 'bool'), ('b2', 'bool'), ('b40', 'bool')) f(k, t);
-- Three searchable fields on the large tenant's type (S2 is specified at 400k).
INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, kind, state, progress)
SELECT md5('tenant0')::uuid, md5('fld' || md5('tenant0')::uuid || md5('ed0/0')::uuid || k || 'search')::uuid,
       md5('ed0/0')::uuid, k, 'text', 'search', 'ready', 100
FROM unnest(ARRAY['title', 'summary', 'notes']) k;

INSERT INTO record_field_values (tenant_id, field_id, record_id, ord, value_text, value_num, value_ts, value_bool)
SELECT r.tenant_id, f.id, r.id, 0, c.value_text, c.value_num, c.value_ts, c.value_bool
FROM records r
JOIN record_index_fields f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id AND f.kind = 'filter'
CROSS JOIN LATERAL record_field_coerce(r.data -> f.field_key, f.value_type) c
WHERE num_nonnulls(c.value_text, c.value_num, c.value_ts, c.value_bool) = 1
ORDER BY r.tenant_id, r.id, f.id;

INSERT INTO record_search_terms (tenant_id, field_id, term, record_id)
SELECT r.tenant_id, f.id, l.lexeme, r.id
FROM records r
JOIN record_index_fields f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id AND f.kind = 'search'
CROSS JOIN LATERAL record_search_lexemes(r.data -> f.field_key) l(lexeme);

SET ROLE bench_owner;
CREATE INDEX rfv_text ON record_field_values (tenant_id, field_id, value_text, record_id) WHERE value_text IS NOT NULL;
CREATE INDEX rfv_num  ON record_field_values (tenant_id, field_id, value_num,  record_id) WHERE value_num  IS NOT NULL;
CREATE INDEX rfv_ts   ON record_field_values (tenant_id, field_id, value_ts,   record_id) WHERE value_ts   IS NOT NULL;
CREATE INDEX rfv_bool ON record_field_values (tenant_id, field_id, value_bool, record_id) WHERE value_bool IS NOT NULL;
ALTER TABLE record_field_values
    ADD CONSTRAINT record_field_values_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    ADD CONSTRAINT record_field_values_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id);
ALTER TABLE record_search_terms
    ADD CONSTRAINT record_search_terms_record_fk FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id),
    ADD CONSTRAINT record_search_terms_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id);
-- D5: AFTER INSERT OR UPDATE OF data, plus BEFORE DELETE.
CREATE TRIGGER record_field_values_sync AFTER INSERT OR UPDATE OF data ON records
    FOR EACH ROW EXECUTE FUNCTION record_field_values_sync();
CREATE TRIGGER record_field_values_purge BEFORE DELETE ON records
    FOR EACH ROW EXECUTE FUNCTION record_field_values_purge();
RESET ROLE;
VACUUM (ANALYZE) record_index_fields;
VACUUM (ANALYZE) record_field_values;
VACUUM (ANALYZE) record_search_terms;

-- Field ids per load actor, for pgbench.
CREATE TABLE bench_actor_f AS
SELECT a.kind, a.n, a.tenant_id AS t, a.user_id AS me, a.groups AS g,
       max(f.id::text) FILTER (WHERE f.field_key = 'tier') AS f_tier,
       max(f.id::text) FILTER (WHERE f.field_key = 'zone') AS f_zone,
       max(f.id::text) FILTER (WHERE f.field_key = 'score') AS f_score,
       max(f.id::text) FILTER (WHERE f.field_key = 'opened') AS f_opened,
       max(f.id::text) FILTER (WHERE f.field_key = 'b001') AS f_b001,
       max(f.id::text) FILTER (WHERE f.field_key = 'b025') AS f_b025,
       max(f.id::text) FILTER (WHERE f.field_key = 'b2') AS f_b2,
       max(f.id::text) FILTER (WHERE f.field_key = 'b40') AS f_b40,
       max(f.id::text) FILTER (WHERE f.field_key = 'summary' AND f.kind = 'search') AS f_summary,
       max(f.id::text) FILTER (WHERE f.field_key = 'notes' AND f.kind = 'search') AS f_notes,
       max(f.id::text) FILTER (WHERE f.field_key = 'title' AND f.kind = 'search') AS f_title
FROM bench_actor a
JOIN record_index_fields f ON f.tenant_id = a.tenant_id
    AND f.entity_definition_id = CASE WHEN a.kind = 'large' THEN md5('ed0/0')::uuid
                                      ELSE md5('ed' || a.n || '/0')::uuid END
GROUP BY 1, 2, 3, 4, 5;
ALTER TABLE bench_actor_f ADD PRIMARY KEY (kind, n);
GRANT SELECT ON bench_actor_f TO nava_app;

SELECT f.kind, f.field_key, count(*) AS rows
FROM record_field_values v JOIN record_index_fields f ON f.tenant_id = v.tenant_id AND f.id = v.field_id
WHERE v.tenant_id = md5('tenant0')::uuid GROUP BY 1, 2 ORDER BY 1, 2;
SELECT count(*) AS terms_large_tenant, count(DISTINCT term) AS distinct_terms FROM record_search_terms;
SELECT c.relname, pg_size_pretty(pg_relation_size(c.oid)) AS size, pg_relation_size(c.oid) AS bytes
FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i')
  AND (c.relname LIKE 'record%' OR c.relname LIKE 'rfv%')
ORDER BY 1;
