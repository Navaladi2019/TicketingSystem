-- ADR-010-A1 (TICK-971): insert cost of the format-2 CHECK + tenant-id
-- trigger at the SP-2 caps (8, 16, 64, 128, 128) with the A1 32 KiB gate on
-- octet_length(rules::text). Run by check32k.sh after TICK-629's functions
-- and the SP-2 prototype (docs/data/acl-format2-caps/format2_check.sql).
-- Directory shape and timing method are SP-2's (check_bench.sql): tenant 0
-- has 5,000 users, 50 groups and 200 org units; 399 small tenants.
\pset pager off
SET max_parallel_workers_per_gather = 0;
SELECT setseed(0.971);

CREATE TABLE groups (tenant_id uuid, id uuid, deleted_at timestamptz, PRIMARY KEY (tenant_id, id));
CREATE TABLE principals (tenant_id uuid, id uuid, deleted_at timestamptz, PRIMARY KEY (tenant_id, id));
CREATE TABLE org_units (tenant_id uuid, id uuid, deleted_at timestamptz, PRIMARY KEY (tenant_id, id));
INSERT INTO principals SELECT md5('tenant' || t)::uuid, md5('u' || t || '/' || n)::uuid, NULL
FROM generate_series(0, 399) t, generate_series(0, CASE WHEN t = 0 THEN 4999 ELSE 19 END) n;
INSERT INTO groups SELECT md5('tenant' || t)::uuid, md5('g' || t || '/' || n)::uuid, NULL
FROM generate_series(0, 399) t, generate_series(0, 49) n;
INSERT INTO org_units SELECT md5('tenant' || t)::uuid, md5('o' || t || '/' || n)::uuid, NULL
FROM generate_series(0, 399) t, generate_series(0, CASE WHEN t = 0 THEN 199 ELSE 9 END) n;
VACUUM ANALYZE groups;
VACUUM ANALYZE principals;
VACUUM ANALYZE org_units;

CREATE TABLE acl_versions (
    tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL, version int NOT NULL,
    rules jsonb NOT NULL, masks jsonb NOT NULL, published_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, entity_definition_id, version));

-- The A1 CHECK: the 32 KiB gate first (so an oversize document is never
-- walked), then the walk at the SP-2 caps. Format 1 is unchanged.
CREATE OR REPLACE FUNCTION acl2_rules_valid(rules jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    IF jsonb_typeof(rules) = 'object' AND rules ? 'format' THEN
        IF octet_length(rules::text) > 32768 THEN RETURN false; END IF;
        RETURN public.acl2_problem(rules, 8, 128, 64, 128) IS NULL;
    END IF;
    RETURN public.acl_rules_valid(rules);
END
$$;

-- k random live ids of tenant 0 (DISTINCT, so the trigger looks up k ids).
CREATE FUNCTION ids(kind text, k int, nmax int) RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(md5(kind || '0/' || x)::uuid) FROM (
        SELECT x FROM generate_series(0, nmax - 1) x ORDER BY random() LIMIT k) s);
CREATE FUNCTION leaf(i int, k int) RETURNS jsonb LANGUAGE sql VOLATILE RETURN (CASE i % 7
    WHEN 0 THEN jsonb_build_object('member_of', jsonb_build_object('groups', ids('g', least(k, 50), 50), 'users', ids('u', k, 5000)))
    WHEN 1 THEN '{"cmp": {"column": "state", "op": "in", "values": ["new", "in_progress", "on_hold"]}}'::jsonb
    WHEN 2 THEN '{"cmp": {"column": "assignment_group_id", "op": "in", "caller": "my_groups"}}'::jsonb
    WHEN 3 THEN '{"cmp": {"column": "requester_id", "op": "eq", "caller": "self"}}'::jsonb
    WHEN 4 THEN jsonb_build_object('cmp', jsonb_build_object('column', 'assignee_id', 'op', 'in', 'values', ids('u', k, 5000)))
    WHEN 5 THEN '{"shared": true}'::jsonb
    ELSE jsonb_build_object('under', jsonb_build_object('of', ids('o', 1, 200))) END);
CREATE FUNCTION wrap(e jsonb, levels int) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
    FOR i IN 1 .. levels LOOP e := jsonb_build_object('all', jsonb_build_array(e)); END LOOP;
    RETURN e;
END $$;
CREATE FUNCTION sz(d jsonb) RETURNS int LANGUAGE sql IMMUTABLE RETURN octet_length(d::text);

-- At the SP-2 leaf caps, written out: 8 policies, each any-of-16 leaves
-- under 6 wrappers (depth 8, 16 children), 128 leaves in the document.
CREATE FUNCTION doc_sp2(k int) RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE pols jsonb := '[]'; e jsonb;
BEGIN
    FOR p IN 1 .. 8 LOOP
        SELECT jsonb_build_object('any', jsonb_agg(leaf(p * 16 + b, k))) INTO e FROM generate_series(0, 15) b;
        pols := pols || jsonb_build_array(jsonb_build_object('name', 'p' || p, 'ops', '["read", "write"]'::jsonb, 'when', wrap(e, 6)));
    END LOOP;
    RETURN jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies', pols);
END $$;
-- The largest k (ids per member_of / assignee leaf) that stays <= 32 KiB.
CREATE FUNCTION doc_sp2_ids_32k() RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE k int := 1; d jsonb; best jsonb;
BEGIN
    LOOP
        d := doc_sp2(k);
        EXIT WHEN sz(d) > 32768;
        best := d; k := k + 1;
    END LOOP;
    RETURN best;
END $$;
-- Pad base with unused conditions (built by mk(i)) to just under 32 KiB.
CREATE FUNCTION pad(base jsonb, kind text) RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE conds jsonb := '[]'; c jsonb; i int := 0; d jsonb := base;
BEGIN
    LOOP
        i := i + 1;
        c := CASE kind
            -- any of 16 mixed leaves, 1 id each
            WHEN 'mixed' THEN jsonb_build_object('name', 'u' || i, 'expr', jsonb_build_object('any',
                (SELECT jsonb_agg(leaf(i * 16 + b, 1)) FROM generate_series(0, 15) b)))
            -- the most nodes per byte: any of 16 {"shared": true}
            WHEN 'nodes' THEN jsonb_build_object('name', 'u' || i, 'expr', jsonb_build_object('any',
                (SELECT jsonb_agg('{"shared": true}'::jsonb) FROM generate_series(1, 16))))
            -- the most refs per byte: any of 16 refs to the previous condition
            ELSE jsonb_build_object('name', 'u' || i, 'expr', CASE WHEN i = 1 THEN '{"shared": true}'::jsonb ELSE jsonb_build_object('any',
                (SELECT jsonb_agg(jsonb_build_object('ref', 'u' || (i - 1))) FROM generate_series(1, 16))) END) END;
        EXIT WHEN sz(d || jsonb_build_object('conditions', conds || jsonb_build_array(c))) > 32768;
        conds := conds || jsonb_build_array(c);
    END LOOP;
    RETURN base || jsonb_build_object('conditions', conds);
END $$;
-- One policy whose single member_of carries as many distinct users as fit:
-- the most ids for the trigger to look up per byte.
CREATE FUNCTION doc_ids_dense_32k() RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE k int := 700; d jsonb; best jsonb;
BEGIN
    LOOP
        d := jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies', jsonb_build_array(
            jsonb_build_object('name', 'p', 'ops', '["read"]'::jsonb, 'when',
                jsonb_build_object('member_of', jsonb_build_object('users', ids('u', k, 5000))))));
        EXIT WHEN sz(d) > 32768;
        best := d; k := k + 5;
    END LOOP;
    RETURN best;
END $$;
-- Refused by the gate: the SP-2 caps document padded to the 1 MiB body limit.
CREATE FUNCTION doc_1mib() RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE d jsonb := doc_sp2(1); conds jsonb := '[]'; i int := 0;
BEGIN
    WHILE sz(d || jsonb_build_object('conditions', conds)) < 1000000 LOOP
        FOR j IN 1 .. 20 LOOP
            i := i + 1;
            conds := conds || jsonb_build_array(jsonb_build_object('name', 'u' || i, 'expr', jsonb_build_object('any',
                (SELECT jsonb_agg(leaf(i * 16 + b, 2)) FROM generate_series(0, 15) b))));
        END LOOP;
    END LOOP;
    RETURN d || jsonb_build_object('conditions', conds);
END $$;
CREATE FUNCTION doc_f1() RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(jsonb_build_object('id', 'rule-' || i, 'ops', '["read", "write"]'::jsonb,
        'who', jsonb_build_object('groups', ids('g', 2, 50), 'users', ids('u', 10, 5000), 'requester', true, 'shared', true),
        'where', jsonb_build_array('{"column": "state", "op": "in", "values": ["new", "in_progress", "on_hold"]}'::jsonb,
                                   '{"column": "assignment_group_id", "op": "in", "caller": "my_groups"}'::jsonb)))
    FROM generate_series(1, 6) i);

CREATE TABLE pool (variant text, i int, rules jsonb);
INSERT INTO pool SELECT 'a f1_6rules', i, doc_f1() FROM generate_series(1, 30) i;
INSERT INTO pool SELECT 'b sp2_caps_k1', i, doc_sp2(1) FROM generate_series(1, 30) i;
INSERT INTO pool SELECT 'c sp2_caps_ids_32k', i, doc_sp2_ids_32k() FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'd sp2_caps_unused_32k', i, pad(doc_sp2(1), 'mixed') FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'e nodes_32k', i, pad(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    '[{"name": "p", "ops": ["read"], "when": {"shared": true}}]'::jsonb), 'nodes') FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'f refs_32k', i, pad(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    '[{"name": "p", "ops": ["read"], "when": {"shared": true}}]'::jsonb), 'refs') FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'g ids_dense_32k', i, doc_ids_dense_32k() FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'z 1mib_refused', i, doc_1mib() FROM generate_series(1, 5) i;

\echo '== documents: text bytes, stored bytes, ids, problem at the SP-2 caps (NULL = valid), A1 CHECK'
SELECT variant, count(*) docs, max(sz(rules)) max_text_bytes, round(avg(pg_column_size(rules))) avg_stored_bytes,
       round(avg((SELECT count(DISTINCT x) FROM jsonb_path_query(rules, 'lax $.**.users[*]') x)
               + (SELECT count(DISTINCT x) FROM jsonb_path_query(rules, 'lax $.** ? (@.column == "assignee_id").values[*]') x)
               + (SELECT count(DISTINCT x) FROM jsonb_path_query(rules, 'lax $.**.groups[*]') x))) ids_per_doc,
       min(CASE WHEN rules ? 'format' THEN coalesce(acl2_problem(rules, 8, 128, 64, 128), 'valid') END) problem,
       bool_and(acl2_rules_valid(rules)) a1_check
FROM pool GROUP BY variant ORDER BY variant;

CREATE TABLE t_ins (variant text, setup text, ms float8);
CREATE FUNCTION time_inserts(setup text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE r record; t0 timestamptz; v int := 0;
BEGIN
    FOR rep IN 1 .. 3 LOOP
        FOR r IN SELECT * FROM pool ORDER BY variant, i LOOP
            v := v + 1;
            t0 := clock_timestamp();
            BEGIN
                INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
                VALUES (md5('tenant0')::uuid, md5(r.variant)::uuid, v, r.rules, '[]');
            EXCEPTION WHEN check_violation THEN NULL;  -- the 1 MiB refusal
            END;
            INSERT INTO t_ins VALUES (r.variant, setup, extract(epoch FROM clock_timestamp() - t0) * 1000);
        END LOOP;
    END LOOP;
END $$;
SELECT time_inserts('1 none');
TRUNCATE acl_versions;
ALTER TABLE acl_versions ADD CONSTRAINT acl_versions_rules_check CHECK (acl2_rules_valid(rules));
SELECT time_inserts('2 check');
TRUNCATE acl_versions;
CREATE TRIGGER acl_versions_tenant_ids AFTER INSERT ON acl_versions FOR EACH ROW EXECUTE FUNCTION acl2_tenant_ids();
SELECT time_inserts('3 check+trigger');
\echo '== single-row insert, ms (3 passes over the pool)'
SELECT variant, setup, count(*) n, round(percentile_cont(0.5) WITHIN GROUP (ORDER BY ms)::numeric, 2) p50,
       round(percentile_cont(0.95) WITHIN GROUP (ORDER BY ms)::numeric, 2) p95, round(max(ms)::numeric, 2) max
FROM t_ins GROUP BY variant, setup ORDER BY variant, setup;

\echo '== the gate alone: octet_length(rules::text) per call, ms'
CREATE TABLE t_gate (variant text, ms float8);
DO $$
DECLARE r record; t0 timestamptz; n int;
BEGIN
    FOR rep IN 1 .. 3 LOOP
        FOR r IN SELECT * FROM pool ORDER BY variant, i LOOP
            t0 := clock_timestamp();
            n := octet_length(r.rules::text);
            INSERT INTO t_gate VALUES (r.variant, extract(epoch FROM clock_timestamp() - t0) * 1000);
        END LOOP;
    END LOOP;
END $$;
SELECT variant, round(percentile_cont(0.5) WITHIN GROUP (ORDER BY ms)::numeric, 3) p50,
       round(max(ms)::numeric, 3) max FROM t_gate GROUP BY variant ORDER BY variant;

\echo '== gate boundary: 32,768 bytes is valid, 32,769 is refused'
WITH base AS (SELECT jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    '[{"name": "p", "ops": ["read"], "when": {"shared": true}}]'::jsonb) d),
pad AS (SELECT d, 32768 - sz(d || '{"conditions": [{"name": "x", "expr": {"cmp": {"column": "state", "op": "eq", "values": [""]}}}]}')
        AS room FROM base)
SELECT n_bytes, acl2_rules_valid(doc) a1_check FROM (
    SELECT sz(x) n_bytes, x doc FROM pad, LATERAL (VALUES
        (d || jsonb_build_object('conditions', jsonb_build_array(jsonb_build_object('name', 'x', 'expr',
            jsonb_build_object('cmp', jsonb_build_object('column', 'state', 'op', 'eq', 'values', jsonb_build_array(repeat('a', room)))))))),
        (d || jsonb_build_object('conditions', jsonb_build_array(jsonb_build_object('name', 'x', 'expr',
            jsonb_build_object('cmp', jsonb_build_object('column', 'state', 'op', 'eq', 'values', jsonb_build_array(repeat('a', room + 1))))))))
    ) v(x)) s ORDER BY n_bytes;
