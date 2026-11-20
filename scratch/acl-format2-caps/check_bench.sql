-- SP-2 (TICK-933) items 1-2: insert cost of the format-2 CHECK and the
-- tenant-id trigger (docs/data/acl-format2-caps/format2_check.sql) on
-- documents at the D4 caps. Run by check.sh. Directory shape of
-- bench/records-list (P3-01): tenant 0 has 5,000 users, 50 groups and 200
-- org units; 399 small tenants have 20 users, 50 groups and 10 units.
\pset pager off
SET max_parallel_workers_per_gather = 0;
SELECT setseed(0.933);

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

-- k random live ids of tenant 0 as a jsonb array.
CREATE FUNCTION ids(kind text, k int, nmax int) RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(md5(kind || '0/' || floor(random() * nmax)::int)::uuid) FROM generate_series(1, k));
-- Leaf i: the leaf kinds in rotation; member_of carries k groups and k users,
-- an assignee cmp k literal users.
CREATE FUNCTION leaf(i int, k int) RETURNS jsonb LANGUAGE sql VOLATILE RETURN (CASE i % 7
    WHEN 0 THEN jsonb_build_object('member_of', jsonb_build_object('groups', ids('g', k, 50), 'users', ids('u', k, 5000)))
    WHEN 1 THEN '{"cmp": {"column": "state", "op": "in", "values": ["new", "in_progress", "on_hold"]}}'::jsonb
    WHEN 2 THEN '{"cmp": {"column": "assignment_group_id", "op": "in", "caller": "my_groups"}}'::jsonb
    WHEN 3 THEN '{"cmp": {"column": "requester_id", "op": "eq", "caller": "self"}}'::jsonb
    WHEN 4 THEN jsonb_build_object('cmp', jsonb_build_object('column', 'assignee_id', 'op', 'in', 'values', ids('u', k, 5000)))
    WHEN 5 THEN '{"shared": true}'::jsonb
    ELSE jsonb_build_object('under', jsonb_build_object('of', ids('o', 1, 200))) END);
-- levels single-child all nodes around e.
CREATE FUNCTION wrap(e jsonb, levels int) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
    FOR i IN 1 .. levels LOOP
        e := jsonb_build_object('all', jsonb_build_array(e));
    END LOOP;
    RETURN e;
END $$;

-- The document at every cap at once, written out (no refs): 64 policies;
-- policy 1 has 256 leaves (all of 16 x any of 16) at depth 8, with 16-child
-- nodes; policies 2-13 have 13 leaves and 14-64 have 12, each at depth 8;
-- 1,024 leaves in all. Every node is written, so the walk visits each one.
CREATE FUNCTION doc_caps(k int) RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE pols jsonb := '[]'; e jsonb; n int;
BEGIN
    SELECT jsonb_build_object('all', jsonb_agg(jsonb_build_object('any',
        (SELECT jsonb_agg(leaf(a * 16 + b, k)) FROM generate_series(0, 15) b)))) INTO e FROM generate_series(0, 15) a;
    pols := pols || jsonb_build_array(jsonb_build_object('name', 'p1', 'ops', '["read"]'::jsonb, 'when', wrap(e, 5)));
    FOR p IN 2 .. 64 LOOP
        n := CASE WHEN p <= 13 THEN 13 ELSE 12 END;
        SELECT jsonb_build_object('all', jsonb_build_array(jsonb_build_object('any', jsonb_agg(leaf(p * 16 + b, k))))) INTO e
        FROM generate_series(1, n) b;
        pols := pols || jsonb_build_array(jsonb_build_object('name', 'p' || p, 'ops', '["read", "write"]'::jsonb, 'when', wrap(e, 5)));
    END LOOP;
    RETURN jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies', pols);
END $$;

-- The same caps through named conditions: leaf_0..6, any_12..16 = any of n
-- refs, c256 = all of 16 refs to any_16; policy 1 refs c256 under 5
-- wrappers; policies 2-64 ref any_13 or any_12 under 6.
CREATE FUNCTION doc_refs(k int) RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE conds jsonb := '[]'; pols jsonb := '[]';
BEGIN
    FOR i IN 0 .. 6 LOOP
        conds := conds || jsonb_build_array(jsonb_build_object('name', 'leaf_' || i, 'expr', leaf(i, k)));
    END LOOP;
    FOR n IN 12 .. 16 LOOP
        conds := conds || jsonb_build_array(jsonb_build_object('name', 'any_' || n, 'expr', jsonb_build_object('any',
            (SELECT jsonb_agg(jsonb_build_object('ref', 'leaf_' || (b % 7))) FROM generate_series(1, n) b))));
    END LOOP;
    conds := conds || jsonb_build_array(jsonb_build_object('name', 'c256', 'expr', jsonb_build_object('all',
        (SELECT jsonb_agg('{"ref": "any_16"}'::jsonb) FROM generate_series(1, 16)))));
    pols := pols || jsonb_build_array(jsonb_build_object('name', 'p1', 'ops', '["read"]'::jsonb, 'when', wrap('{"ref": "c256"}', 5)));
    FOR p IN 2 .. 64 LOOP
        pols := pols || jsonb_build_array(jsonb_build_object('name', 'p' || p, 'ops', '["read"]'::jsonb,
            'when', wrap(jsonb_build_object('ref', 'any_' || CASE WHEN p <= 13 THEN 13 ELSE 12 END), 6)));
    END LOOP;
    RETURN jsonb_build_object('format', 2, 'conditions', conds, 'policies', pols);
END $$;

-- Adversarial, refused: a doubling chain d0 = leaf, d_i = all[ref d_{i-1},
-- ref d_{i-1}] up to d60 (2^60 leaves if expanded); one policy refs d60.
CREATE FUNCTION doc_doubling() RETURNS jsonb LANGUAGE sql VOLATILE RETURN jsonb_build_object('format', 2,
    'conditions', '[{"name": "d0", "expr": {"shared": true}}]'::jsonb || (SELECT jsonb_agg(jsonb_build_object('name', 'd' || i,
        'expr', jsonb_build_object('all', jsonb_build_array(jsonb_build_object('ref', 'd' || (i - 1)), jsonb_build_object('ref', 'd' || (i - 1))))))
        FROM generate_series(1, 60) i),
    'policies', '[{"name": "p", "ops": ["read"], "when": {"ref": "d60"}}]'::jsonb);

-- Adversarial, accepted: within every cap, padded to the 1 MiB request
-- limit (internal/platform/idempotency maxBodySize) with unused conditions,
-- each any of 16 leaves with 2 ids per member_of.
CREATE FUNCTION doc_1mib() RETURNS jsonb LANGUAGE plpgsql VOLATILE AS $$
DECLARE d jsonb := doc_caps(1); conds jsonb := '[]'; i int := 0;
BEGIN
    WHILE octet_length((d || jsonb_build_object('conditions', conds))::text) < 1000000 LOOP
        FOR j IN 1 .. 20 LOOP
            i := i + 1;
            conds := conds || jsonb_build_array(jsonb_build_object('name', 'u' || i, 'expr', jsonb_build_object('any',
                (SELECT jsonb_agg(leaf(i * 16 + b, 2)) FROM generate_series(0, 15) b))));
        END LOOP;
    END LOOP;
    RETURN d || jsonb_build_object('conditions', conds);
END $$;

-- Format-1 baseline: TICK-629's realistic 6-rule policy (2 groups, 10 users).
CREATE FUNCTION doc_f1() RETURNS jsonb LANGUAGE sql VOLATILE RETURN (
    SELECT jsonb_agg(jsonb_build_object('id', 'rule-' || i, 'ops', '["read", "write"]'::jsonb,
        'who', jsonb_build_object('groups', ids('g', 2, 50), 'users', ids('u', 10, 5000), 'requester', true, 'shared', true),
        'where', jsonb_build_array('{"column": "state", "op": "in", "values": ["new", "in_progress", "on_hold"]}'::jsonb,
                                   '{"column": "assignment_group_id", "op": "in", "caller": "my_groups"}'::jsonb)))
    FROM generate_series(1, 6) i);

-- Distinct documents of each variant (fresh random ids each). caps_k55 puts
-- 55 ids in every member_of and assignee cmp leaf: about 1 MiB and most of
-- tenant 0's 5,000 users. Today's rules put no cap on ids per leaf.
CREATE TABLE pool (variant text, i int, rules jsonb);
INSERT INTO pool SELECT 'f1_6rules', i, doc_f1() FROM generate_series(1, 50) i;
INSERT INTO pool SELECT 'caps_k1', i, doc_caps(1) FROM generate_series(1, 50) i;
INSERT INTO pool SELECT 'caps_k10', i, doc_caps(10) FROM generate_series(1, 50) i;
INSERT INTO pool SELECT 'caps_refs_k10', i, doc_refs(10) FROM generate_series(1, 50) i;
INSERT INTO pool SELECT 'caps_k55_1mib', i, doc_caps(55) FROM generate_series(1, 20) i;
INSERT INTO pool SELECT 'caps_unused_1mib', i, doc_1mib() FROM generate_series(1, 20) i;
INSERT INTO pool SELECT 'doubling_refused', i, doc_doubling() FROM generate_series(1, 50) i;

\echo '== documents: size and problem code (NULL = valid)'
SELECT variant, count(*) docs, round(avg(octet_length(rules::text))) avg_text_bytes,
       round(avg(pg_column_size(rules))) avg_stored_bytes,
       min(coalesce(acl2_problem(rules), 'valid')) problem
FROM pool WHERE rules ? 'format' GROUP BY variant ORDER BY variant;
SELECT variant, count(*) docs, round(avg(octet_length(rules::text))) avg_text_bytes, bool_and(acl2_rules_valid(rules)) valid
FROM pool WHERE NOT rules ? 'format' GROUP BY variant;

-- The predicate alone, per document, clock around each call (the argument
-- is a column, so nothing is constant-folded).
CREATE TABLE t_pred (variant text, ms float8);
DO $$
DECLARE r record; t0 timestamptz; ok boolean;
BEGIN
    FOR rep IN 1 .. 3 LOOP
        FOR r IN SELECT * FROM pool ORDER BY variant, i LOOP
            t0 := clock_timestamp();
            ok := acl2_rules_valid(r.rules);
            INSERT INTO t_pred VALUES (r.variant, extract(epoch FROM clock_timestamp() - t0) * 1000);
        END LOOP;
    END LOOP;
END $$;
\echo '== CHECK predicate alone (acl2_rules_valid), ms per document, 3 passes'
SELECT variant, count(*) n, round(percentile_cont(0.5) WITHIN GROUP (ORDER BY ms)::numeric, 3) p50,
       round(percentile_cont(0.95) WITHIN GROUP (ORDER BY ms)::numeric, 3) p95, round(max(ms)::numeric, 3) max
FROM t_pred GROUP BY variant ORDER BY p50;

\echo '== distinct ids per document (what the trigger looks up)'
SELECT variant, round(avg(cardinality(g))) groups, round(avg(cardinality(u))) users, round(avg(cardinality(o))) org_units FROM (
    SELECT variant,
        ARRAY(SELECT DISTINCT x FROM (SELECT jsonb_path_query(rules, 'lax $.**.groups[*]') #>> '{}' x
              UNION ALL SELECT jsonb_path_query(rules, 'lax $.** ? (@.column == "assignment_group_id" && exists(@.values)).values[*]') #>> '{}') s) g,
        ARRAY(SELECT DISTINCT x FROM (SELECT jsonb_path_query(rules, 'lax $.**.users[*]') #>> '{}' x
              UNION ALL SELECT jsonb_path_query(rules, 'lax $.** ? ((@.column == "requester_id" || @.column == "assignee_id") && exists(@.values)).values[*]') #>> '{}') s) u,
        ARRAY(SELECT DISTINCT jsonb_path_query(rules, 'lax $.**.under.of ? (@.type() == "array")[*]') #>> '{}') o
    FROM pool WHERE variant <> 'doubling_refused') z
GROUP BY variant ORDER BY variant;

-- Single-row inserts, one statement each, timed around the statement:
-- before (no CHECK, no trigger), CHECK only, CHECK + trigger.
CREATE TABLE t_ins (variant text, setup text, ms float8);
CREATE FUNCTION time_inserts(setup text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE r record; t0 timestamptz; v int := 0;
BEGIN
    FOR rep IN 1 .. 3 LOOP
        FOR r IN SELECT * FROM pool WHERE variant <> 'doubling_refused' ORDER BY variant, i LOOP
            v := v + 1;
            t0 := clock_timestamp();
            INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks)
            VALUES (md5('tenant0')::uuid, md5(r.variant)::uuid, v, r.rules, '[]');
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
SELECT variant, setup, count(*) n, round(percentile_cont(0.5) WITHIN GROUP (ORDER BY ms)::numeric, 3) p50,
       round(percentile_cont(0.95) WITHIN GROUP (ORDER BY ms)::numeric, 3) p95, round(max(ms)::numeric, 3) max
FROM t_ins GROUP BY variant, setup ORDER BY variant, setup;

\echo '== refusals at insert: SQLSTATE, constraint or detail'
DO $$
DECLARE st text; dt text; cn text;
BEGIN
    BEGIN
        INSERT INTO acl_versions VALUES (md5('tenant0')::uuid, gen_random_uuid(), 1,
            (SELECT rules FROM pool WHERE variant = 'doubling_refused' LIMIT 1), '[]');
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS st = RETURNED_SQLSTATE, cn = CONSTRAINT_NAME;
        RAISE NOTICE 'doubling chain: SQLSTATE % constraint % code %', st, cn,
            acl2_problem((SELECT rules FROM pool WHERE variant = 'doubling_refused' LIMIT 1));
    END;
    BEGIN
        -- tenant 1 publishing tenant 0's ids
        INSERT INTO acl_versions VALUES (md5('tenant1')::uuid, gen_random_uuid(), 1,
            (SELECT rules FROM pool WHERE variant = 'caps_k1' LIMIT 1), '[]');
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS st = RETURNED_SQLSTATE, dt = PG_EXCEPTION_DETAIL;
        RAISE NOTICE 'foreign tenant ids: SQLSTATE % detail %', st, dt;
    END;
END $$;

\echo '== cap boundaries: at the cap is valid, one over is refused with its code'
SELECT 'depth 8' c, acl2_problem(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    jsonb_build_array(jsonb_build_object('name', 'p', 'ops', '["read"]'::jsonb, 'when', wrap('{"shared": true}', 7))))) code
UNION ALL SELECT 'depth 9', acl2_problem(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    jsonb_build_array(jsonb_build_object('name', 'p', 'ops', '["read"]'::jsonb, 'when', wrap('{"shared": true}', 8)))))
UNION ALL SELECT 'depth 9 through a ref', acl2_problem(jsonb_build_object('format', 2,
    'conditions', jsonb_build_array(jsonb_build_object('name', 'c', 'expr', wrap('{"shared": true}', 4))), 'policies',
    jsonb_build_array(jsonb_build_object('name', 'p', 'ops', '["read"]'::jsonb, 'when', wrap('{"ref": "c"}', 4)))))
UNION ALL SELECT '17 children', acl2_problem(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    jsonb_build_array(jsonb_build_object('name', 'p', 'ops', '["read"]'::jsonb, 'when',
        jsonb_build_object('any', (SELECT jsonb_agg('{"shared": true}'::jsonb) FROM generate_series(1, 17)))))))
UNION ALL SELECT '257 leaves through refs', acl2_problem(jsonb_build_object('format', 2,
    'conditions', jsonb_build_array('{"name": "s", "expr": {"shared": true}}'::jsonb,
        jsonb_build_object('name', 'a', 'expr', jsonb_build_object('any', (SELECT jsonb_agg('{"ref": "s"}'::jsonb) FROM generate_series(1, 16)))),
        jsonb_build_object('name', 'b', 'expr', jsonb_build_object('all', (SELECT jsonb_agg('{"ref": "a"}'::jsonb) FROM generate_series(1, 16))))),
    'policies', '[{"name": "p", "ops": ["read"], "when": {"any": [{"ref": "b"}, {"shared": true}]}}]'::jsonb))
UNION ALL SELECT '65 policies', acl2_problem(jsonb_build_object('format', 2, 'conditions', '[]'::jsonb, 'policies',
    (SELECT jsonb_agg(jsonb_build_object('name', 'p' || i, 'ops', '["read"]'::jsonb, 'when', '{"shared": true}'::jsonb)) FROM generate_series(1, 65) i)))
UNION ALL SELECT '1,025 leaves (caps doc, last policy 13)', acl2_problem(jsonb_set((SELECT rules FROM pool WHERE variant = 'caps_k1' LIMIT 1),
    '{policies,63,when}', wrap(jsonb_build_object('any', (SELECT jsonb_agg('{"shared": true}'::jsonb) FROM generate_series(1, 13))), 6)))
UNION ALL SELECT 'self ref', acl2_problem('{"format": 2, "conditions": [{"name": "a", "expr": {"not": {"ref": "a"}}}], "policies": []}')
UNION ALL SELECT 'a-b-a cycle', acl2_problem('{"format": 2, "conditions": [{"name": "a", "expr": {"ref": "b"}}, {"name": "b", "expr": {"ref": "a"}}], "policies": []}')
UNION ALL SELECT 'missing ref', acl2_problem('{"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"ref": "x"}}]}')
UNION ALL SELECT 'fn node', acl2_problem('{"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"fn": "now"}}]}')
UNION ALL SELECT 'two keys', acl2_problem('{"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"shared": true, "not": {"shared": true}}}]}')
UNION ALL SELECT 'duplicate policy', acl2_problem('{"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"shared": true}}, {"name": "p", "ops": ["read"], "when": {"shared": true}}]}')
UNION ALL SELECT 'bad name', acl2_problem('{"format": 2, "conditions": [], "policies": [{"name": "P-1", "ops": ["read"], "when": {"shared": true}}]}')
UNION ALL SELECT 'self on state', acl2_problem('{"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"cmp": {"column": "state", "op": "eq", "caller": "self"}}}]}');
