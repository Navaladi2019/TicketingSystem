-- TICK-1059 (S31 follow-up, TODO-S31-01; SP-1d change 5 on TICK-946,
-- ADR-010-A2): the database CHECK learns the under-set cap that D4 enforces
-- at publish since TICK-1034. A format-2 policy names at most 4 distinct
-- under start sets U after its refs expand, the value of authz.MaxUnderSets.
-- A stored document above it is refused by acl_versions_rules_check with
-- {"column": "rules", "code": "under_sets_exceeded", "pointer":
-- "/policies/<i>"}, the pointer and code the route reports.
--
-- One U, as authz.underSets counts it: {"of": "caller"} is one; a listed
-- {"of": [ids]} is one per SET of ids, so order, repeats and letter case do
-- not make a new one (acl2_under_key: distinct lower-case ids, sorted, comma
-- joined). Refs expand: each condition's set is computed once, in the Kahn
-- order acl2_problem already walks, so a doubling chain still costs its
-- written size. Sets saturate one above the cap (acl2_under_union), so a
-- document with thousands of under leaves costs a handful of comparisons per
-- leaf, not a quadratic merge.
--
-- The cap is written once, in acl2_max_under_sets(); the saturation point is
-- derived from it. Raise it only together with authz.MaxUnderSets: the
-- integration test TestTICK1059UnderSetsCap pins both at 4.
--
-- Not checked here, as before: under_nested (a unit and its ancestor in one
-- list). It reads org_unit_closure, which a CHECK cannot.
--
-- Online: catalogue changes only, in one transaction; no table is scanned and
-- no lock beyond the functions' own is taken. CHECKs are not re-run on stored
-- rows. A row stored above the cap before this migration stays; the planner
-- already voids such a policy (TICK-1034), and none exists outside story
-- branches, since under ships with S31.

-- +goose Up

-- authz.MaxUnderSets. One definition, read by acl2_under_union and
-- acl2_problem.
-- +goose StatementBegin
CREATE FUNCTION acl2_max_under_sets() RETURNS int
    LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$ SELECT 4 $$;
-- +goose StatementEnd

-- The key of one under U, given its "of" value (already shape-checked):
-- 'caller', or the distinct lower-case ids sorted and joined with ','. The
-- "C" collation makes the order byte order, independent of the database's.
-- PL/pgSQL, so the query's plan is cached across calls, and one id (the
-- common case) needs no query at all.
-- +goose StatementBegin
CREATE FUNCTION acl2_under_key(o jsonb) RETURNS text
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    IF jsonb_typeof(o) = 'string' THEN RETURN 'caller'; END IF;
    IF jsonb_array_length(o) = 1 THEN RETURN lower(o ->> 0); END IF;
    RETURN (SELECT string_agg(DISTINCT lower(x) COLLATE "C", ',' ORDER BY lower(x) COLLATE "C")
            FROM jsonb_array_elements_text(o) AS t(x));
END
$$;
-- +goose StatementEnd

-- The union of two key sets, each ';'-joined ('' is empty; keys hold no
-- ';'). It stops growing at acl2_max_under_sets() + 1 keys: that is enough
-- to know the cap is passed, and bounds every later merge.
-- +goose StatementBegin
CREATE FUNCTION acl2_under_union(a text, b text) RETURNS text
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    sat CONSTANT int := public.acl2_max_under_sets() + 1;
    r text[];
    k text;
BEGIN
    IF b = '' THEN RETURN a; END IF;
    IF a = '' AND cardinality(string_to_array(b, ';')) <= sat THEN RETURN b; END IF;
    r := string_to_array(a, ';');
    FOREACH k IN ARRAY string_to_array(b, ';') LOOP
        EXIT WHEN cardinality(r) >= sat;
        IF NOT k = ANY (r) THEN r := r || k; END IF;
    END LOOP;
    RETURN array_to_string(r, ';');
END
$$;
-- +goose StatementEnd

-- acl2_expr gains OUT unders, so it is dropped and created again (CREATE OR
-- REPLACE cannot change OUT parameters). Nothing records a dependency on it:
-- acl2_problem calls it by name at run time, and both change in this
-- transaction.
DROP FUNCTION acl2_expr(jsonb, text);

-- +goose StatementBegin
CREATE FUNCTION acl2_expr(e jsonb, ptr text,
    OUT problem jsonb, OUT leaves bigint, OUT depth int, OUT refs text[], OUT ref_ptrs text[], OUT ref_depths int[],
    OUT unders text)
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    st  jsonb[] := ARRAY[e];
    sd  int[]   := ARRAY[1];
    sp  text[]  := ARRAY[ptr];
    top int     := 1;
    n   jsonb;
    d   int;
    p   text;
    k   text;
    kp  text;
    v   jsonb;
    len int;
    nr  int := 0;
BEGIN
    leaves := 0; depth := 0; refs := '{}'; ref_ptrs := '{}'; ref_depths := '{}'; unders := '';
    WHILE top > 0 LOOP
        n := st[top]; d := sd[top]; p := sp[top]; top := top - 1;
        IF jsonb_typeof(n) IS DISTINCT FROM 'object' THEN
            problem := public.acl2_bad('type', p); RETURN;
        END IF;
        k := CASE WHEN n ? 'cmp' THEN 'cmp' WHEN n ? 'all' THEN 'all' WHEN n ? 'any' THEN 'any'
                  WHEN n ? 'ref' THEN 'ref' WHEN n ? 'not' THEN 'not' WHEN n ? 'member_of' THEN 'member_of'
                  WHEN n ? 'shared' THEN 'shared' WHEN n ? 'under' THEN 'under' END;
        IF k IS NULL OR n - k <> '{}'::jsonb THEN
            -- None or two keys: unknown_key at the node. One key outside D3:
            -- unknown_key at that key (authz node()).
            IF k IS NULL AND n <> '{}'::jsonb AND (SELECT count(*) FROM jsonb_object_keys(n)) = 1 THEN
                problem := public.acl2_unknown_key(n, p, '{}');
            ELSE
                problem := public.acl2_bad('unknown_key', p);
            END IF;
            RETURN;
        END IF;
        v := n -> k;
        kp := p || '/' || k;
        CASE k
        WHEN 'all', 'any' THEN
            IF jsonb_typeof(v) <> 'array' THEN problem := public.acl2_bad('type', kp); RETURN; END IF;
            len := jsonb_array_length(v);
            IF len < 1 OR len > 16 THEN problem := public.acl2_bad('child_count', kp); RETURN; END IF;
            FOR i IN REVERSE len - 1 .. 0 LOOP
                top := top + 1; st[top] := v -> i; sd[top] := d + 1; sp[top] := kp || '/' || i;
            END LOOP;
        WHEN 'not' THEN
            top := top + 1; st[top] := v; sd[top] := d + 1; sp[top] := kp;
        WHEN 'ref' THEN
            IF jsonb_typeof(v) <> 'string' THEN problem := public.acl2_bad('type', kp); RETURN; END IF;
            IF NOT public.acl2_name_ok(v #>> '{}') THEN problem := public.acl2_bad('invalid_name', kp); RETURN; END IF;
            nr := nr + 1; refs[nr] := v #>> '{}'; ref_ptrs[nr] := kp; ref_depths[nr] := d;
        WHEN 'member_of' THEN
            problem := public.acl2_member_of_problem(v, kp);
            IF problem IS NOT NULL THEN RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'cmp' THEN
            IF NOT public.acl_cond_valid(v) THEN
                problem := coalesce(public.acl2_cond_problem(v, kp), public.acl2_bad('value', kp)); RETURN;
            END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'shared' THEN
            IF jsonb_typeof(v) <> 'boolean' THEN problem := public.acl2_bad('type', kp); RETURN; END IF;
            IF v <> 'true'::jsonb THEN problem := public.acl2_bad('value', kp); RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'under' THEN
            problem := public.acl2_under_problem(v, kp);
            IF problem IS NOT NULL THEN RETURN; END IF;
            unders := public.acl2_under_union(unders, public.acl2_under_key(v -> 'of'));
            leaves := leaves + 1; depth := greatest(depth, d);
        END CASE;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl2_problem(doc jsonb) RETURNS jsonb
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    -- A1-1, not configurable: authz's MaxDepth, MaxPolicies,
    -- MaxPolicyLeaves, MaxDocumentLeaves (MaxChildren is in acl2_expr).
    max_depth         CONSTANT int := 8;
    max_policies      CONSTANT int := 64;
    max_policy_leaves CONSTANT int := 128;
    max_doc_leaves    CONSTANT int := 128;
    -- SP-1d change 5 (TICK-946): authz's MaxUnderSets, written once.
    max_under_sets    CONSTANT int := public.acl2_max_under_sets();
    sat               CONSTANT bigint := 1000000000;
    conds jsonb; pols jsonb; c jsonb; p jsonb; ptr text;
    nc int; np int;
    idx jsonb;                    -- condition name -> 1-based index
    pnames text[] := '{}';        -- policy names so far (at most 64)
    w record;
    lv bigint[]; dp int[];        -- per condition: local counts, then totals
    outdeg int[];
    -- edges: condition efrom refs ename (index eto) at depth ed, pointer eptr
    efrom int[] := '{}'; ename text[] := '{}'; eto int[] := '{}'; ed int[] := '{}'; eptr text[] := '{}';
    ne int := 0;
    rstart int[]; rorder int[]; fill int[];
    q int[]; qh int := 1; qt int := 0;
    acc_l bigint[]; acc_d int[];
    j int; f int; e int;
    pl bigint; pd int; tot bigint := 0;
    nm text; dup bigint;
    us text[]; acc_u text[];     -- per condition: under keys, then with refs
    pu text;
BEGIN
    IF jsonb_typeof(doc) IS DISTINCT FROM 'object' THEN
        RETURN public.acl2_bad('type', '');
    END IF;
    IF doc - ARRAY['format', 'conditions', 'policies'] <> '{}'::jsonb THEN
        RETURN public.acl2_unknown_key(doc, '', ARRAY['format', 'conditions', 'policies']);
    END IF;
    IF NOT doc ? 'format' THEN RETURN public.acl2_bad('required', '/format'); END IF;
    IF jsonb_typeof(doc -> 'format') <> 'number' THEN RETURN public.acl2_bad('type', '/format'); END IF;
    IF doc -> 'format' <> '2'::jsonb THEN RETURN public.acl2_bad('value', '/format'); END IF;
    IF NOT doc ? 'conditions' THEN RETURN public.acl2_bad('required', '/conditions'); END IF;
    IF jsonb_typeof(doc -> 'conditions') <> 'array' THEN RETURN public.acl2_bad('type', '/conditions'); END IF;
    IF NOT doc ? 'policies' THEN RETURN public.acl2_bad('required', '/policies'); END IF;
    IF jsonb_typeof(doc -> 'policies') <> 'array' THEN RETURN public.acl2_bad('type', '/policies'); END IF;
    conds := doc -> 'conditions'; pols := doc -> 'policies';
    nc := jsonb_array_length(conds); np := jsonb_array_length(pols);
    IF np > max_policies THEN
        RETURN public.acl2_bad('policies_exceeded', '/policies');
    END IF;

    -- Each condition once: shape, name, its expression's local counts and
    -- its refs.
    lv := array_fill(0::bigint, ARRAY[nc]); dp := array_fill(0, ARRAY[nc]); outdeg := array_fill(0, ARRAY[nc]);
    us := array_fill(''::text, ARRAY[nc]);
    FOR i IN 1 .. nc LOOP
        c := conds -> (i - 1); ptr := '/conditions/' || (i - 1);
        IF jsonb_typeof(c) <> 'object' THEN RETURN public.acl2_bad('type', ptr); END IF;
        IF c - ARRAY['name', 'expr'] <> '{}'::jsonb THEN
            RETURN public.acl2_unknown_key(c, ptr, ARRAY['name', 'expr']);
        END IF;
        IF NOT c ? 'name' THEN RETURN public.acl2_bad('required', ptr || '/name'); END IF;
        IF jsonb_typeof(c -> 'name') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/name'); END IF;
        IF NOT public.acl2_name_ok(c ->> 'name') THEN RETURN public.acl2_bad('invalid_name', ptr || '/name'); END IF;
        IF NOT c ? 'expr' THEN RETURN public.acl2_bad('required', ptr || '/expr'); END IF;
        w := public.acl2_expr(c -> 'expr', ptr || '/expr');
        IF w.problem IS NOT NULL THEN RETURN w.problem; END IF;
        lv[i] := w.leaves; dp[i] := w.depth; us[i] := w.unders;
        FOR k IN 1 .. coalesce(array_length(w.refs, 1), 0) LOOP
            ne := ne + 1; efrom[ne] := i; ename[ne] := w.refs[k]; ed[ne] := w.ref_depths[k]; eptr[ne] := w.ref_ptrs[k];
            outdeg[i] := outdeg[i] + 1;
        END LOOP;
    END LOOP;

    -- Names unique (the second one is the duplicate, as authz reports it),
    -- then name -> index, each in one query: growing a jsonb object one name
    -- at a time copies it each time, and only bytes bound the conditions.
    IF nc > 0 THEN
        SELECT min(o) INTO dup FROM (
            SELECT o, row_number() OVER (PARTITION BY ce ->> 'name' ORDER BY o) rn
            FROM jsonb_array_elements(conds) WITH ORDINALITY AS t(ce, o)) s
        WHERE rn = 2;
        IF dup IS NOT NULL THEN
            RETURN public.acl2_bad('duplicate_name', '/conditions/' || (dup - 1) || '/name');
        END IF;
        SELECT jsonb_object_agg(ce ->> 'name', o) INTO idx
        FROM jsonb_array_elements(conds) WITH ORDINALITY AS t(ce, o);
    ELSE
        idx := '{}';
    END IF;
    FOR k IN 1 .. ne LOOP
        IF NOT idx ? ename[k] THEN RETURN public.acl2_bad('unresolved_ref', eptr[k]); END IF;
        eto[k] := (idx ->> ename[k])::int;
    END LOOP;

    -- Kahn: a condition is done when every condition it refs is done.
    IF nc > 0 THEN
        rstart := array_fill(0, ARRAY[nc + 1]); fill := array_fill(0, ARRAY[nc]);
        FOR k IN 1 .. ne LOOP rstart[eto[k]] := rstart[eto[k]] + 1; END LOOP;
        j := 1;
        FOR i IN 1 .. nc LOOP f := rstart[i]; rstart[i] := j; j := j + f; END LOOP;
        rstart[nc + 1] := j;
        rorder := array_fill(0, ARRAY[greatest(ne, 1)]);
        FOR k IN 1 .. ne LOOP
            rorder[rstart[eto[k]] + fill[eto[k]]] := k; fill[eto[k]] := fill[eto[k]] + 1;
        END LOOP;
        acc_l := array_fill(0::bigint, ARRAY[nc]); acc_d := array_fill(0, ARRAY[nc]);
        acc_u := array_fill(''::text, ARRAY[nc]);
        q := array_fill(0, ARRAY[nc]);
        FOR i IN 1 .. nc LOOP
            IF outdeg[i] = 0 THEN qt := qt + 1; q[qt] := i; END IF;
        END LOOP;
        WHILE qh <= qt LOOP
            j := q[qh]; qh := qh + 1;
            FOR k IN rstart[j] .. rstart[j + 1] - 1 LOOP
                e := rorder[k]; f := efrom[e];
                acc_l[f] := least(sat, acc_l[f] + lv[j]);
                acc_d[f] := greatest(acc_d[f], ed[e] + dp[j] - 1);
                acc_u[f] := public.acl2_under_union(acc_u[f], us[j]);
                outdeg[f] := outdeg[f] - 1;
                IF outdeg[f] = 0 THEN
                    lv[f] := least(sat, lv[f] + acc_l[f]); dp[f] := greatest(dp[f], acc_d[f]);
                    us[f] := public.acl2_under_union(us[f], acc_u[f]);
                    qt := qt + 1; q[qt] := f;
                END IF;
            END LOOP;
        END LOOP;
        IF qt < nc THEN
            -- The first ref, in document order, from a waiting condition to a
            -- waiting one: on the cycle, or on the way to it.
            FOR k IN 1 .. ne LOOP
                IF outdeg[efrom[k]] > 0 AND outdeg[eto[k]] > 0 THEN
                    RETURN public.acl2_bad('ref_cycle', eptr[k]);
                END IF;
            END LOOP;
        END IF;
    END IF;

    -- Policies: shape, then expanded counts against the caps.
    FOR i IN 1 .. np LOOP
        p := pols -> (i - 1); ptr := '/policies/' || (i - 1);
        IF jsonb_typeof(p) <> 'object' THEN RETURN public.acl2_bad('type', ptr); END IF;
        IF p - ARRAY['name', 'ops', 'when'] <> '{}'::jsonb THEN
            RETURN public.acl2_unknown_key(p, ptr, ARRAY['name', 'ops', 'when']);
        END IF;
        IF NOT p ? 'name' THEN RETURN public.acl2_bad('required', ptr || '/name'); END IF;
        IF jsonb_typeof(p -> 'name') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/name'); END IF;
        nm := p ->> 'name';
        IF NOT public.acl2_name_ok(nm) THEN RETURN public.acl2_bad('invalid_name', ptr || '/name'); END IF;
        IF nm = ANY (pnames) THEN RETURN public.acl2_bad('duplicate_name', ptr || '/name'); END IF;
        pnames := pnames || nm;
        IF NOT p ? 'ops' THEN RETURN public.acl2_bad('required', ptr || '/ops'); END IF;
        IF jsonb_typeof(p -> 'ops') <> 'array' THEN RETURN public.acl2_bad('type', ptr || '/ops'); END IF;
        IF p -> 'ops' = '[]'::jsonb THEN RETURN public.acl2_bad('empty', ptr || '/ops'); END IF;
        FOR k IN 0 .. jsonb_array_length(p -> 'ops') - 1 LOOP
            IF jsonb_typeof(p -> 'ops' -> k) <> 'string' THEN
                RETURN public.acl2_bad('type', ptr || '/ops/' || k);
            END IF;
            IF p -> 'ops' ->> k NOT IN ('read', 'write') THEN
                RETURN public.acl2_bad('value', ptr || '/ops/' || k);
            END IF;
        END LOOP;
        IF NOT p ? 'when' THEN RETURN public.acl2_bad('required', ptr || '/when'); END IF;
        w := public.acl2_expr(p -> 'when', ptr || '/when');
        IF w.problem IS NOT NULL THEN RETURN w.problem; END IF;
        pl := w.leaves; pd := w.depth; pu := w.unders;
        FOR k IN 1 .. coalesce(array_length(w.refs, 1), 0) LOOP
            IF NOT idx ? w.refs[k] THEN RETURN public.acl2_bad('unresolved_ref', w.ref_ptrs[k]); END IF;
            j := (idx ->> w.refs[k])::int;
            pl := least(sat, pl + lv[j]); pd := greatest(pd, w.ref_depths[k] + dp[j] - 1);
            pu := public.acl2_under_union(pu, us[j]);
        END LOOP;
        IF pd > max_depth THEN RETURN public.acl2_bad('depth_exceeded', ptr || '/when'); END IF;
        IF pl > max_policy_leaves THEN RETURN public.acl2_bad('policy_leaves_exceeded', ptr || '/when'); END IF;
        tot := tot + pl;
        IF tot > max_doc_leaves THEN RETURN public.acl2_bad('document_leaves_exceeded', '/policies'); END IF;
        IF cardinality(string_to_array(pu, ';')) > max_under_sets THEN
            RETURN public.acl2_bad('under_sets_exceeded', ptr);
        END IF;
    END LOOP;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

-- +goose Down

-- TICK-940's acl2_expr and acl2_problem, as 20261119100000 created them.
DROP FUNCTION acl2_expr(jsonb, text);

-- +goose StatementBegin
CREATE FUNCTION acl2_expr(e jsonb, ptr text,
    OUT problem jsonb, OUT leaves bigint, OUT depth int, OUT refs text[], OUT ref_ptrs text[], OUT ref_depths int[])
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    st  jsonb[] := ARRAY[e];
    sd  int[]   := ARRAY[1];
    sp  text[]  := ARRAY[ptr];
    top int     := 1;
    n   jsonb;
    d   int;
    p   text;
    k   text;
    kp  text;
    v   jsonb;
    len int;
    nr  int := 0;
BEGIN
    leaves := 0; depth := 0; refs := '{}'; ref_ptrs := '{}'; ref_depths := '{}';
    WHILE top > 0 LOOP
        n := st[top]; d := sd[top]; p := sp[top]; top := top - 1;
        IF jsonb_typeof(n) IS DISTINCT FROM 'object' THEN
            problem := public.acl2_bad('type', p); RETURN;
        END IF;
        k := CASE WHEN n ? 'cmp' THEN 'cmp' WHEN n ? 'all' THEN 'all' WHEN n ? 'any' THEN 'any'
                  WHEN n ? 'ref' THEN 'ref' WHEN n ? 'not' THEN 'not' WHEN n ? 'member_of' THEN 'member_of'
                  WHEN n ? 'shared' THEN 'shared' WHEN n ? 'under' THEN 'under' END;
        IF k IS NULL OR n - k <> '{}'::jsonb THEN
            -- None or two keys: unknown_key at the node. One key outside D3:
            -- unknown_key at that key (authz node()).
            IF k IS NULL AND n <> '{}'::jsonb AND (SELECT count(*) FROM jsonb_object_keys(n)) = 1 THEN
                problem := public.acl2_unknown_key(n, p, '{}');
            ELSE
                problem := public.acl2_bad('unknown_key', p);
            END IF;
            RETURN;
        END IF;
        v := n -> k;
        kp := p || '/' || k;
        CASE k
        WHEN 'all', 'any' THEN
            IF jsonb_typeof(v) <> 'array' THEN problem := public.acl2_bad('type', kp); RETURN; END IF;
            len := jsonb_array_length(v);
            IF len < 1 OR len > 16 THEN problem := public.acl2_bad('child_count', kp); RETURN; END IF;
            FOR i IN REVERSE len - 1 .. 0 LOOP
                top := top + 1; st[top] := v -> i; sd[top] := d + 1; sp[top] := kp || '/' || i;
            END LOOP;
        WHEN 'not' THEN
            top := top + 1; st[top] := v; sd[top] := d + 1; sp[top] := kp;
        WHEN 'ref' THEN
            IF jsonb_typeof(v) <> 'string' THEN problem := public.acl2_bad('type', kp); RETURN; END IF;
            IF NOT public.acl2_name_ok(v #>> '{}') THEN problem := public.acl2_bad('invalid_name', kp); RETURN; END IF;
            nr := nr + 1; refs[nr] := v #>> '{}'; ref_ptrs[nr] := kp; ref_depths[nr] := d;
        WHEN 'member_of' THEN
            problem := public.acl2_member_of_problem(v, kp);
            IF problem IS NOT NULL THEN RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'cmp' THEN
            IF NOT public.acl_cond_valid(v) THEN
                problem := coalesce(public.acl2_cond_problem(v, kp), public.acl2_bad('value', kp)); RETURN;
            END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'shared' THEN
            IF jsonb_typeof(v) <> 'boolean' THEN problem := public.acl2_bad('type', kp); RETURN; END IF;
            IF v <> 'true'::jsonb THEN problem := public.acl2_bad('value', kp); RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'under' THEN
            problem := public.acl2_under_problem(v, kp);
            IF problem IS NOT NULL THEN RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        END CASE;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl2_problem(doc jsonb) RETURNS jsonb
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    -- A1-1, not configurable: authz's MaxDepth, MaxPolicies,
    -- MaxPolicyLeaves, MaxDocumentLeaves (MaxChildren is in acl2_expr).
    max_depth         CONSTANT int := 8;
    max_policies      CONSTANT int := 64;
    max_policy_leaves CONSTANT int := 128;
    max_doc_leaves    CONSTANT int := 128;
    sat               CONSTANT bigint := 1000000000;
    conds jsonb; pols jsonb; c jsonb; p jsonb; ptr text;
    nc int; np int;
    idx jsonb;                    -- condition name -> 1-based index
    pnames text[] := '{}';        -- policy names so far (at most 64)
    w record;
    lv bigint[]; dp int[];        -- per condition: local counts, then totals
    outdeg int[];
    -- edges: condition efrom refs ename (index eto) at depth ed, pointer eptr
    efrom int[] := '{}'; ename text[] := '{}'; eto int[] := '{}'; ed int[] := '{}'; eptr text[] := '{}';
    ne int := 0;
    rstart int[]; rorder int[]; fill int[];
    q int[]; qh int := 1; qt int := 0;
    acc_l bigint[]; acc_d int[];
    j int; f int; e int;
    pl bigint; pd int; tot bigint := 0;
    nm text; dup bigint;
BEGIN
    IF jsonb_typeof(doc) IS DISTINCT FROM 'object' THEN
        RETURN public.acl2_bad('type', '');
    END IF;
    IF doc - ARRAY['format', 'conditions', 'policies'] <> '{}'::jsonb THEN
        RETURN public.acl2_unknown_key(doc, '', ARRAY['format', 'conditions', 'policies']);
    END IF;
    IF NOT doc ? 'format' THEN RETURN public.acl2_bad('required', '/format'); END IF;
    IF jsonb_typeof(doc -> 'format') <> 'number' THEN RETURN public.acl2_bad('type', '/format'); END IF;
    IF doc -> 'format' <> '2'::jsonb THEN RETURN public.acl2_bad('value', '/format'); END IF;
    IF NOT doc ? 'conditions' THEN RETURN public.acl2_bad('required', '/conditions'); END IF;
    IF jsonb_typeof(doc -> 'conditions') <> 'array' THEN RETURN public.acl2_bad('type', '/conditions'); END IF;
    IF NOT doc ? 'policies' THEN RETURN public.acl2_bad('required', '/policies'); END IF;
    IF jsonb_typeof(doc -> 'policies') <> 'array' THEN RETURN public.acl2_bad('type', '/policies'); END IF;
    conds := doc -> 'conditions'; pols := doc -> 'policies';
    nc := jsonb_array_length(conds); np := jsonb_array_length(pols);
    IF np > max_policies THEN
        RETURN public.acl2_bad('policies_exceeded', '/policies');
    END IF;

    -- Each condition once: shape, name, its expression's local counts and
    -- its refs.
    lv := array_fill(0::bigint, ARRAY[nc]); dp := array_fill(0, ARRAY[nc]); outdeg := array_fill(0, ARRAY[nc]);
    FOR i IN 1 .. nc LOOP
        c := conds -> (i - 1); ptr := '/conditions/' || (i - 1);
        IF jsonb_typeof(c) <> 'object' THEN RETURN public.acl2_bad('type', ptr); END IF;
        IF c - ARRAY['name', 'expr'] <> '{}'::jsonb THEN
            RETURN public.acl2_unknown_key(c, ptr, ARRAY['name', 'expr']);
        END IF;
        IF NOT c ? 'name' THEN RETURN public.acl2_bad('required', ptr || '/name'); END IF;
        IF jsonb_typeof(c -> 'name') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/name'); END IF;
        IF NOT public.acl2_name_ok(c ->> 'name') THEN RETURN public.acl2_bad('invalid_name', ptr || '/name'); END IF;
        IF NOT c ? 'expr' THEN RETURN public.acl2_bad('required', ptr || '/expr'); END IF;
        w := public.acl2_expr(c -> 'expr', ptr || '/expr');
        IF w.problem IS NOT NULL THEN RETURN w.problem; END IF;
        lv[i] := w.leaves; dp[i] := w.depth;
        FOR k IN 1 .. coalesce(array_length(w.refs, 1), 0) LOOP
            ne := ne + 1; efrom[ne] := i; ename[ne] := w.refs[k]; ed[ne] := w.ref_depths[k]; eptr[ne] := w.ref_ptrs[k];
            outdeg[i] := outdeg[i] + 1;
        END LOOP;
    END LOOP;

    -- Names unique (the second one is the duplicate, as authz reports it),
    -- then name -> index, each in one query: growing a jsonb object one name
    -- at a time copies it each time, and only bytes bound the conditions.
    IF nc > 0 THEN
        SELECT min(o) INTO dup FROM (
            SELECT o, row_number() OVER (PARTITION BY ce ->> 'name' ORDER BY o) rn
            FROM jsonb_array_elements(conds) WITH ORDINALITY AS t(ce, o)) s
        WHERE rn = 2;
        IF dup IS NOT NULL THEN
            RETURN public.acl2_bad('duplicate_name', '/conditions/' || (dup - 1) || '/name');
        END IF;
        SELECT jsonb_object_agg(ce ->> 'name', o) INTO idx
        FROM jsonb_array_elements(conds) WITH ORDINALITY AS t(ce, o);
    ELSE
        idx := '{}';
    END IF;
    FOR k IN 1 .. ne LOOP
        IF NOT idx ? ename[k] THEN RETURN public.acl2_bad('unresolved_ref', eptr[k]); END IF;
        eto[k] := (idx ->> ename[k])::int;
    END LOOP;

    -- Kahn: a condition is done when every condition it refs is done.
    IF nc > 0 THEN
        rstart := array_fill(0, ARRAY[nc + 1]); fill := array_fill(0, ARRAY[nc]);
        FOR k IN 1 .. ne LOOP rstart[eto[k]] := rstart[eto[k]] + 1; END LOOP;
        j := 1;
        FOR i IN 1 .. nc LOOP f := rstart[i]; rstart[i] := j; j := j + f; END LOOP;
        rstart[nc + 1] := j;
        rorder := array_fill(0, ARRAY[greatest(ne, 1)]);
        FOR k IN 1 .. ne LOOP
            rorder[rstart[eto[k]] + fill[eto[k]]] := k; fill[eto[k]] := fill[eto[k]] + 1;
        END LOOP;
        acc_l := array_fill(0::bigint, ARRAY[nc]); acc_d := array_fill(0, ARRAY[nc]);
        q := array_fill(0, ARRAY[nc]);
        FOR i IN 1 .. nc LOOP
            IF outdeg[i] = 0 THEN qt := qt + 1; q[qt] := i; END IF;
        END LOOP;
        WHILE qh <= qt LOOP
            j := q[qh]; qh := qh + 1;
            FOR k IN rstart[j] .. rstart[j + 1] - 1 LOOP
                e := rorder[k]; f := efrom[e];
                acc_l[f] := least(sat, acc_l[f] + lv[j]);
                acc_d[f] := greatest(acc_d[f], ed[e] + dp[j] - 1);
                outdeg[f] := outdeg[f] - 1;
                IF outdeg[f] = 0 THEN
                    lv[f] := least(sat, lv[f] + acc_l[f]); dp[f] := greatest(dp[f], acc_d[f]);
                    qt := qt + 1; q[qt] := f;
                END IF;
            END LOOP;
        END LOOP;
        IF qt < nc THEN
            -- The first ref, in document order, from a waiting condition to a
            -- waiting one: on the cycle, or on the way to it.
            FOR k IN 1 .. ne LOOP
                IF outdeg[efrom[k]] > 0 AND outdeg[eto[k]] > 0 THEN
                    RETURN public.acl2_bad('ref_cycle', eptr[k]);
                END IF;
            END LOOP;
        END IF;
    END IF;

    -- Policies: shape, then expanded counts against the caps.
    FOR i IN 1 .. np LOOP
        p := pols -> (i - 1); ptr := '/policies/' || (i - 1);
        IF jsonb_typeof(p) <> 'object' THEN RETURN public.acl2_bad('type', ptr); END IF;
        IF p - ARRAY['name', 'ops', 'when'] <> '{}'::jsonb THEN
            RETURN public.acl2_unknown_key(p, ptr, ARRAY['name', 'ops', 'when']);
        END IF;
        IF NOT p ? 'name' THEN RETURN public.acl2_bad('required', ptr || '/name'); END IF;
        IF jsonb_typeof(p -> 'name') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/name'); END IF;
        nm := p ->> 'name';
        IF NOT public.acl2_name_ok(nm) THEN RETURN public.acl2_bad('invalid_name', ptr || '/name'); END IF;
        IF nm = ANY (pnames) THEN RETURN public.acl2_bad('duplicate_name', ptr || '/name'); END IF;
        pnames := pnames || nm;
        IF NOT p ? 'ops' THEN RETURN public.acl2_bad('required', ptr || '/ops'); END IF;
        IF jsonb_typeof(p -> 'ops') <> 'array' THEN RETURN public.acl2_bad('type', ptr || '/ops'); END IF;
        IF p -> 'ops' = '[]'::jsonb THEN RETURN public.acl2_bad('empty', ptr || '/ops'); END IF;
        FOR k IN 0 .. jsonb_array_length(p -> 'ops') - 1 LOOP
            IF jsonb_typeof(p -> 'ops' -> k) <> 'string' THEN
                RETURN public.acl2_bad('type', ptr || '/ops/' || k);
            END IF;
            IF p -> 'ops' ->> k NOT IN ('read', 'write') THEN
                RETURN public.acl2_bad('value', ptr || '/ops/' || k);
            END IF;
        END LOOP;
        IF NOT p ? 'when' THEN RETURN public.acl2_bad('required', ptr || '/when'); END IF;
        w := public.acl2_expr(p -> 'when', ptr || '/when');
        IF w.problem IS NOT NULL THEN RETURN w.problem; END IF;
        pl := w.leaves; pd := w.depth;
        FOR k IN 1 .. coalesce(array_length(w.refs, 1), 0) LOOP
            IF NOT idx ? w.refs[k] THEN RETURN public.acl2_bad('unresolved_ref', w.ref_ptrs[k]); END IF;
            j := (idx ->> w.refs[k])::int;
            pl := least(sat, pl + lv[j]); pd := greatest(pd, w.ref_depths[k] + dp[j] - 1);
        END LOOP;
        IF pd > max_depth THEN RETURN public.acl2_bad('depth_exceeded', ptr || '/when'); END IF;
        IF pl > max_policy_leaves THEN RETURN public.acl2_bad('policy_leaves_exceeded', ptr || '/when'); END IF;
        tot := tot + pl;
        IF tot > max_doc_leaves THEN RETURN public.acl2_bad('document_leaves_exceeded', '/policies'); END IF;
    END LOOP;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

DROP FUNCTION acl2_under_union(text, text);
DROP FUNCTION acl2_under_key(jsonb);
DROP FUNCTION acl2_max_under_sets();
