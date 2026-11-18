-- ADR-010-A1 rev 1 (TICK-971): insert cost of the format-2 CHECK, the masks
-- CHECK and the tenant-id trigger at the SP-2 caps (8, 16, 64, 128, 128),
-- with the A1-2 size caps on octet_length(rules::text) and
-- octet_length(masks::text) set to the psql variable :cap (bytes). Run by
-- cap_bench.sh, once at 32,768 (the default) and once at 131,072 (the
-- ceiling), after TICK-629's functions and the SP-2 prototype
-- (docs/data/acl-format2-caps/format2_check.sql).
-- Directory shape and timing method are SP-2's (check_bench.sql): tenant 0
-- has 5,000 users, 50 groups and 200 org units; 399 small tenants.
\pset pager off
SET max_parallel_workers_per_gather = 0;
SELECT setseed(0.971);
SELECT set_config('bench.cap', :'cap', false) AS cap;

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

-- The A1-2a form: above the cap the walk returns TRUE without walking, and
-- acl_versions_bytes_rules_check (added below) is the one constraint that
-- refuses. PostgreSQL checks CHECKs in name order, so acl_versions_rules_check
-- runs first: the early return is what keeps an oversize row from being walked.
CREATE OR REPLACE FUNCTION acl2_rules_valid(rules jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    IF jsonb_typeof(rules) = 'object' AND rules ? 'format' THEN
        IF octet_length(rules::text) > current_setting('bench.cap')::int THEN RETURN true; END IF;
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
        EXIT WHEN sz(d) > current_setting('bench.cap')::int;
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
        EXIT WHEN sz(d || jsonb_build_object('conditions', conds || jsonb_build_array(c))) > current_setting('bench.cap')::int;
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
        EXIT WHEN sz(d) > current_setting('bench.cap')::int;
        best := d; k := k + 5;
    END LOOP;
    RETURN best;
END $$;
-- Refused by the size cap: the SP-2 caps document padded to the 1 MiB body limit.
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

-- Format-1 masks (TICK-629 shape) with as many ids as fit under the cap:
-- n fields, each read by k users and 10 groups and written by k users.
CREATE FUNCTION masks_doc(n int, k int) RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(jsonb_build_object('field', 'f' || i,
        'read',  jsonb_build_object('users', ids('u', k, 5000), 'groups', ids('g', 10, 50)),
        'write', jsonb_build_object('users', ids('u', k, 5000))))
    FROM generate_series(1, n) i);
CREATE FUNCTION masks_cap() RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE n int := 1; d jsonb; best jsonb;
BEGIN
    LOOP
        d := masks_doc(n, 90);
        EXIT WHEN sz(d) > current_setting('bench.cap')::int;
        best := d; n := n + 1;
    END LOOP;
    RETURN best;
END $$;
CREATE TABLE pool (variant text, i int, rules jsonb, masks jsonb NOT NULL DEFAULT '[]');
INSERT INTO pool SELECT 'a f1_6rules', i, doc_f1() FROM generate_series(1, 30) i;
INSERT INTO pool SELECT 'b sp2_caps_k1', i, doc_sp2(1) FROM generate_series(1, 30) i;
INSERT INTO pool SELECT 'c sp2_caps_ids_cap', i, doc_sp2_ids_32k() FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'd sp2_caps_unused_cap', i, pad(doc_sp2(1), 'mixed') FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'e nodes_cap', i, pad(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    '[{"name": "p", "ops": ["read"], "when": {"shared": true}}]'::jsonb), 'nodes') FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'f refs_cap', i, pad(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    '[{"name": "p", "ops": ["read"], "when": {"shared": true}}]'::jsonb), 'refs') FROM generate_series(1, 10) i;
INSERT INTO pool SELECT 'g ids_dense_cap', i, doc_ids_dense_32k() FROM generate_series(1, 10) i;
INSERT INTO pool (variant, i, rules, masks) SELECT 'h masks_cap', i, doc_sp2(1), masks_cap() FROM generate_series(1, 10) i;
-- The worst row: the worst rules shape (f) and masks at the cap together.
INSERT INTO pool (variant, i, rules, masks) SELECT 'i refs_cap+masks_cap', i, rules, masks_cap()
FROM pool WHERE variant = 'f refs_cap';
INSERT INTO pool SELECT 'z 1mib_refused', i, doc_1mib() FROM generate_series(1, 5) i;
INSERT INTO pool (variant, i, rules, masks) SELECT 'z masks_1mib_refused', i, doc_sp2(1), masks_doc(128, 95) FROM generate_series(1, 5) i;

\echo '== documents: text bytes, stored bytes, ids, problem at the SP-2 caps (NULL = valid), A1 CHECK'
SELECT variant, count(*) docs, max(sz(rules)) max_text_bytes, max(sz(masks)) masks_bytes, round(avg(pg_column_size(rules))) avg_stored_bytes,
       round(avg((SELECT count(DISTINCT x) FROM jsonb_path_query(rules, 'lax $.**.users[*]') x)
               + (SELECT count(DISTINCT x) FROM jsonb_path_query(rules, 'lax $.** ? (@.column == "assignee_id").values[*]') x)
               + (SELECT count(DISTINCT x) FROM jsonb_path_query(rules, 'lax $.**.groups[*]') x))) ids_per_doc,
       min(CASE WHEN rules ? 'format' THEN coalesce(acl2_problem(rules, 8, 128, 64, 128), 'valid') END) problem,
       bool_and(acl2_rules_valid(rules)) a1_check
FROM pool GROUP BY variant ORDER BY variant;

CREATE TABLE t_ins (variant text, setup text, ms float8, outcome text);
CREATE FUNCTION time_inserts(setup text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE r record; t0 timestamptz; v int := 0; outcome text;
BEGIN
    FOR rep IN 1 .. 3 LOOP
        FOR r IN SELECT * FROM pool ORDER BY variant, i LOOP
            v := v + 1;
            t0 := clock_timestamp();
            outcome := 'inserted';
            BEGIN
                INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
                VALUES (md5('tenant0')::uuid, md5(r.variant)::uuid, v, r.rules, r.masks);
            EXCEPTION WHEN check_violation THEN
                GET STACKED DIAGNOSTICS outcome = CONSTRAINT_NAME;
                outcome := 'refused ' || coalesce(nullif(outcome, ''), 'trigger');
            END;
            INSERT INTO t_ins VALUES (r.variant, setup, extract(epoch FROM clock_timestamp() - t0) * 1000, outcome);
        END LOOP;
    END LOOP;
END $$;
SELECT time_inserts('1 none');
TRUNCATE acl_versions;
ALTER TABLE acl_versions ADD CONSTRAINT acl_versions_rules_check CHECK (acl2_rules_valid(rules)),
    ADD CONSTRAINT acl_versions_bytes_rules_check CHECK (NOT (rules ? 'format') OR octet_length(rules::text) <= current_setting('bench.cap')::int),
    ADD CONSTRAINT acl_versions_masks_check CHECK (acl_masks_valid(masks)),
    ADD CONSTRAINT acl_versions_bytes_masks_check CHECK (octet_length(masks::text) <= current_setting('bench.cap')::int);
SELECT time_inserts('2 check');
TRUNCATE acl_versions;
CREATE TRIGGER acl_versions_tenant_ids AFTER INSERT ON acl_versions FOR EACH ROW EXECUTE FUNCTION acl2_tenant_ids();
SELECT time_inserts('3 check+trigger');
\echo '== single-row insert, ms (3 passes over the pool)'
SELECT variant, setup, string_agg(DISTINCT outcome, ', ') outcome, count(*) n, round(percentile_cont(0.5) WITHIN GROUP (ORDER BY ms)::numeric, 2) p50,
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

\echo '== size boundary at insert (A1-2a form, all constraints and the trigger in place): cap is inserted, cap + 1 is refused'
CREATE FUNCTION padded(extra int) RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE d jsonb := jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    '[{"name": "p", "ops": ["read"], "when": {"shared": true}}]'::jsonb); room int;
BEGIN
    room := current_setting('bench.cap')::int - sz(d || '{"conditions": [{"name": "x", "expr": {"cmp": {"column": "state", "op": "eq", "values": [""]}}}]}');
    RETURN d || jsonb_build_object('conditions', jsonb_build_array(jsonb_build_object('name', 'x', 'expr',
        jsonb_build_object('cmp', jsonb_build_object('column', 'state', 'op', 'eq', 'values', jsonb_build_array(repeat('a', room + extra)))))));
END $$;
CREATE FUNCTION padded_masks(extra int) RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE room int;
BEGIN
    room := current_setting('bench.cap')::int - sz('[{"field": "", "read": {"users": []}, "write": {}}]'::jsonb);
    RETURN jsonb_build_array(jsonb_build_object('field', repeat('f', room + extra), 'read', '{"users": []}'::jsonb, 'write', '{}'::jsonb));
END $$;
DO $$
DECLARE cn text; v int := 900000;
BEGIN
    FOR extra IN 0 .. 1 LOOP
        FOR col IN 1 .. 2 LOOP
            v := v + 1;
            BEGIN
                INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
                VALUES (md5('tenant0')::uuid, gen_random_uuid(), v,
                        CASE col WHEN 1 THEN padded(extra) ELSE padded(0) - 'conditions' || '{"conditions": []}' END,
                        CASE col WHEN 2 THEN padded_masks(extra) ELSE '[]' END);
                RAISE NOTICE '% at cap + %: inserted', CASE col WHEN 1 THEN 'rules' ELSE 'masks' END, extra;
            EXCEPTION WHEN check_violation THEN
                GET STACKED DIAGNOSTICS cn = CONSTRAINT_NAME;
                RAISE NOTICE '% at cap + %: refused by %', CASE col WHEN 1 THEN 'rules' ELSE 'masks' END, extra, cn;
            END;
        END LOOP;
    END LOOP;
END $$;
