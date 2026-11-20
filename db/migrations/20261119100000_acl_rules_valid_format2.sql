-- TICK-940 (S30 P11-08; ADR-010 D2-D4, amendment A1; PLAN-011 C1, C6,
-- D-11-11, D-11-12): the database backstop for format-2 access documents,
-- the 128 KiB byte ceilings on rules and masks, and acl_versions insert for
-- nava_app only. Figures, counting rules and pointers:
-- docs/data/acl-format2-checks.md.
--
-- 1. acl_versions_rules_check keeps its name and its call,
--    acl_rules_valid(rules). That function is replaced: an object goes to
--    acl2_rules_valid (format 2), an array keeps TICK-629's format-1 check
--    unchanged, so every stored row still passes. acl_uuids_valid gets
--    SP-2's one-regex body (same accepted set, 28x faster at 5,000 ids).
--
-- 2. acl2_rules_valid refuses by raising 23514 on acl_versions_rules_check
--    with a DETAIL the route can read: {"column": "rules", "code": <C1>,
--    "pointer": <JSON pointer into the rules value>}. It names the FIRST
--    problem its walk meets; the route (TICK-939) reports them all. A plain
--    false would carry no detail.
--
-- 3. Caps (A1-1, not configurable): depth 8 after expanding refs, 16
--    children per all/any, 64 policies, 128 leaves per policy and per
--    document. Refs resolve and have no cycle. Unused conditions are shape-
--    and ref-checked but not counted (D4).
--
-- 4. Byte ceilings (A1-2b): acl_versions_bytes_rules_check and
--    acl_versions_bytes_masks_check, 131,072 bytes of the jsonb text form.
--    Their names sort before acl_versions_masks_check and
--    acl_versions_rules_check, and PostgreSQL runs CHECKs in name order, so
--    an oversize row is refused before any walk. acl2_rules_valid also
--    returns true above the ceiling without walking, so that guarantee
--    survives a rename. Added NOT VALID here (catalogue only, no scan) and
--    validated by the next migration under SHARE UPDATE EXCLUSIVE: masks is
--    checked on every row, format 1 included.
--
-- 5. Tenant ids (D4, D-11-11, A1-3 d): acl_versions_tenant_ids, an AFTER
--    INSERT row trigger. AFTER, so the CHECKs run first: the trigger sees
--    only shape-valid values (every id a uuid string) and never scans an
--    oversize row. Every group and user id in rules (format 1 and 2) and
--    masks must be a live row of NEW.tenant_id, filtered explicitly, not
--    only by RLS: the owner bypasses RLS. 23514 with DETAIL {"column",
--    "code": "foreign_id", "pointer"} of the first foreign id. SECURITY
--    INVOKER (I3); it runs as nava_app, which reads principals and groups.
--    Org-unit ids (under) are TICK-947's, when under ships.
--
-- 6. A1-3: the insert policy is nava_app's alone. db/grants.yaml drops
--    nava_worker's INSERT and its EXECUTE on every acl% function, and
--    grants nava_app EXECUTE on the acl2_* functions its CHECKs call.
--
-- Every function is SECURITY INVOKER, calls the others schema-qualified,
-- and uses only pg_catalog operators and built-ins.

-- +goose Up

-- An array of uuid strings: SP-2's body (docs/data/acl-format2-caps.md).
-- One regular expression over the array's canonical jsonb text: jsonb prints
-- ["a", "b"] with ", " between elements, and a uuid string has nothing to
-- escape, so a non-string, nested or malformed element breaks the pattern.
-- No counted repetition ({8}), which costs 6.6 us a match. [] is valid.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl_uuids_valid(a jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    RETURN jsonb_typeof(a) IS NOT DISTINCT FROM 'array' AND a::text ~ ('^\[("'
        || '[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]-'
        || '[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]-[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]-'
        || '[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]-'
        || '[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]'
        || '[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]'
        || '"(, )?)*\]$');
END
$$;
-- +goose StatementEnd

-- A problem: {"code": <C1 code>, "pointer": <JSON pointer>}.
-- +goose StatementBegin
CREATE FUNCTION acl2_bad(code text, pointer text) RETURNS jsonb
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    RETURN pg_catalog.jsonb_build_object('code', code, 'pointer', pointer);
-- +goose StatementEnd

-- A pointer segment (RFC 6901: ~ is ~0, / is ~1), as authz's child() writes.
-- +goose StatementBegin
CREATE FUNCTION acl2_seg(k text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    RETURN '/' || replace(replace(k, '~', '~0'), '/', '~1');
-- +goose StatementEnd

-- The first key of obj outside allowed, as unknown_key at ptr/<key>, or
-- NULL. Only on a refusal path: it is a query.
-- +goose StatementBegin
CREATE FUNCTION acl2_unknown_key(obj jsonb, ptr text, allowed text[]) RETURNS jsonb
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    RETURN (SELECT public.acl2_bad('unknown_key', ptr || public.acl2_seg(k))
            FROM jsonb_object_keys(obj) k WHERE k <> ALL (allowed) ORDER BY k COLLATE "C" LIMIT 1);
-- +goose StatementEnd

-- A valid condition or policy name (D2): ^[a-z][a-z0-9_]{0,62}$, written
-- without counted repetition.
-- +goose StatementBegin
CREATE FUNCTION acl2_name_ok(s text) RETURNS boolean
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    RETURN s ~ '^[a-z][a-z0-9_]*$' AND length(s) <= 63;
-- +goose StatementEnd

-- The first problem of an id array at ptr (type, then each element: type or
-- value), or NULL. The fast path is acl_uuids_valid.
-- +goose StatementBegin
CREATE FUNCTION acl2_ids_problem(a jsonb, ptr text) RETURNS jsonb
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    IF public.acl_uuids_valid(a) THEN
        RETURN NULL;
    END IF;
    IF jsonb_typeof(a) IS DISTINCT FROM 'array' THEN
        RETURN public.acl2_bad('type', ptr);
    END IF;
    FOR i IN 0 .. jsonb_array_length(a) - 1 LOOP
        IF jsonb_typeof(a -> i) <> 'string' THEN
            RETURN public.acl2_bad('type', ptr || '/' || i);
        END IF;
        IF NOT public.acl_uuids_valid(jsonb_build_array(a -> i)) THEN
            RETURN public.acl2_bad('value', ptr || '/' || i);
        END IF;
    END LOOP;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

-- member_of at ptr: {groups?, users?}, uuid arrays, at least one id.
-- +goose StatementBegin
CREATE FUNCTION acl2_member_of_problem(v jsonb, ptr text) RETURNS jsonb
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    p jsonb;
    n int := 0;
BEGIN
    IF jsonb_typeof(v) <> 'object' THEN
        RETURN public.acl2_bad('type', ptr);
    END IF;
    IF v - ARRAY['groups', 'users'] <> '{}'::jsonb THEN
        RETURN public.acl2_unknown_key(v, ptr, ARRAY['groups', 'users']);
    END IF;
    IF v ? 'groups' THEN
        p := public.acl2_ids_problem(v -> 'groups', ptr || '/groups');
        IF p IS NOT NULL THEN RETURN p; END IF;
        n := n + jsonb_array_length(v -> 'groups');
    END IF;
    IF v ? 'users' THEN
        p := public.acl2_ids_problem(v -> 'users', ptr || '/users');
        IF p IS NOT NULL THEN RETURN p; END IF;
        n := n + jsonb_array_length(v -> 'users');
    END IF;
    IF n = 0 THEN
        RETURN public.acl2_bad('empty', ptr);
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

-- cmp at ptr: today's Cond with today's rules, in the order authz's cond()
-- checks them. Only called when acl_cond_valid refused it.
-- +goose StatementBegin
CREATE FUNCTION acl2_cond_problem(c jsonb, ptr text) RETURNS jsonb
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    col    text;
    op     text;
    caller text := '';
    vals   jsonb := '[]';
BEGIN
    IF jsonb_typeof(c) <> 'object' THEN
        RETURN public.acl2_bad('type', ptr);
    END IF;
    IF c - ARRAY['column', 'op', 'values', 'caller'] <> '{}'::jsonb THEN
        RETURN public.acl2_unknown_key(c, ptr, ARRAY['column', 'op', 'values', 'caller']);
    END IF;
    IF NOT c ? 'column' THEN RETURN public.acl2_bad('required', ptr || '/column'); END IF;
    IF jsonb_typeof(c -> 'column') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/column'); END IF;
    col := c ->> 'column';
    IF col NOT IN ('state', 'requester_id', 'assignee_id', 'assignment_group_id') THEN
        RETURN public.acl2_bad('value', ptr || '/column');
    END IF;
    IF NOT c ? 'op' THEN RETURN public.acl2_bad('required', ptr || '/op'); END IF;
    IF jsonb_typeof(c -> 'op') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/op'); END IF;
    op := c ->> 'op';
    IF op NOT IN ('eq', 'in') THEN RETURN public.acl2_bad('value', ptr || '/op'); END IF;
    IF c ? 'caller' THEN
        IF jsonb_typeof(c -> 'caller') <> 'string' THEN RETURN public.acl2_bad('type', ptr || '/caller'); END IF;
        caller := c ->> 'caller';
        IF caller NOT IN ('', 'self', 'my_groups') THEN RETURN public.acl2_bad('value', ptr || '/caller'); END IF;
    END IF;
    IF c ? 'values' THEN
        vals := c -> 'values';
        IF jsonb_typeof(vals) <> 'array' THEN RETURN public.acl2_bad('type', ptr || '/values'); END IF;
        FOR i IN 0 .. jsonb_array_length(vals) - 1 LOOP
            IF jsonb_typeof(vals -> i) <> 'string' THEN
                RETURN public.acl2_bad('type', ptr || '/values/' || i);
            END IF;
        END LOOP;
    END IF;
    IF caller = '' THEN
        IF op = 'eq' AND jsonb_array_length(vals) <> 1 THEN
            RETURN public.acl2_bad('condition', ptr || '/values');
        END IF;
        IF col <> 'state' THEN
            RETURN public.acl2_ids_problem(vals, ptr || '/values');
        END IF;
        RETURN NULL;
    END IF;
    IF vals <> '[]'::jsonb
       OR (caller = 'self' AND col NOT IN ('requester_id', 'assignee_id'))
       OR (caller = 'my_groups' AND col <> 'assignment_group_id') THEN
        RETURN public.acl2_bad('condition', ptr || '/caller');
    END IF;
    RETURN NULL;
END
$$;
-- +goose StatementEnd

-- under at ptr: {"of": "caller"} or {"of": [uuid, ...]} with at least one.
-- +goose StatementBegin
CREATE FUNCTION acl2_under_problem(v jsonb, ptr text) RETURNS jsonb
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    o jsonb;
BEGIN
    IF jsonb_typeof(v) <> 'object' THEN
        RETURN public.acl2_bad('type', ptr);
    END IF;
    IF v - 'of' <> '{}'::jsonb THEN
        RETURN public.acl2_unknown_key(v, ptr, ARRAY['of']);
    END IF;
    IF NOT v ? 'of' THEN
        RETURN public.acl2_bad('required', ptr || '/of');
    END IF;
    o := v -> 'of';
    CASE jsonb_typeof(o)
    WHEN 'string' THEN
        IF o <> '"caller"'::jsonb THEN RETURN public.acl2_bad('value', ptr || '/of'); END IF;
        RETURN NULL;
    WHEN 'array' THEN
        IF o = '[]'::jsonb THEN RETURN public.acl2_bad('empty', ptr || '/of'); END IF;
        RETURN public.acl2_ids_problem(o, ptr || '/of');
    ELSE
        RETURN public.acl2_bad('type', ptr || '/of');
    END CASE;
END
$$;
-- +goose StatementEnd

-- One expression at ptr: its shape, and its counts with refs left symbolic.
-- A worklist, not recursion (a PL/pgSQL call per node costs more than an
-- array slot), children pushed last-first so the first problem met is the
-- first in document order. A lone leaf has depth 1; all, any and not add
-- one. Each ref is returned with its pointer and its own depth, for
-- acl2_problem to resolve. The node-key test is plain expressions, not a
-- query over jsonb_object_keys (an SPI plan per node; SP-2 halved the cost
-- with this). Each leaf's fast path is the TICK-629 check; the *_problem
-- functions run only when it refuses.
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


-- The first problem of a format-2 object, or NULL when it is valid. Order:
-- the top level; the policy count; each condition (shape, name, expr);
-- duplicate condition names; unresolved refs of conditions; cycles; each
-- policy (shape, name, ops, when, refs, then the caps on its expanded
-- counts).
--
-- Counting (SP-2): leaves and depth are memoized per condition in Kahn's
-- order over the reference graph, never expanded, so a doubling chain costs
-- its written size, not 2^n. A ref adds its condition's leaves; its depth is
-- the ref's own depth plus the condition's depth minus one. A condition
-- still waiting when the queue empties is on a cycle or reaches one. Leaf
-- counts saturate, so a doubling chain cannot overflow bigint. Only
-- policies are capped: an unused condition is allowed (D4).
-- +goose StatementBegin
CREATE FUNCTION acl2_problem(doc jsonb) RETURNS jsonb
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

-- The format-2 CHECK predicate. True for a valid document; for an invalid
-- one it raises 23514 on acl_versions_rules_check (the one constraint that
-- calls it, through acl_rules_valid) with the first problem as DETAIL, so
-- the error says where and why. Do not call it outside that CHECK.
--
-- Above 131,072 bytes it returns true WITHOUT walking: that row is refused
-- by acl_versions_bytes_rules_check (ADR-010-A1 A1-2b), which this depends
-- on. Both ship in this migration; dropping or raising that constraint
-- without removing this return lets an oversize document in unchecked.
-- +goose StatementBegin
CREATE FUNCTION acl2_rules_valid(rules jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    p jsonb;
BEGIN
    IF rules ? 'format' AND octet_length(rules::text) > 131072 THEN
        RETURN true;
    END IF;
    p := public.acl2_problem(rules);
    IF p IS NULL THEN
        RETURN true;
    END IF;
    RAISE EXCEPTION 'new row for relation "acl_versions" violates check constraint "acl_versions_rules_check"'
        USING ERRCODE = 'check_violation', CONSTRAINT = 'acl_versions_rules_check',
              SCHEMA = 'public', TABLE = 'acl_versions', COLUMN = 'rules',
              DETAIL = (jsonb_build_object('column', 'rules') || p)::text;
END
$$;
-- +goose StatementEnd

-- acl_versions.rules, the predicate of acl_versions_rules_check: an object
-- is a format-2 document (acl2_rules_valid), an array a format-1 rule list
-- (TICK-629's body, unchanged), anything else a type problem at "".
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl_rules_valid(rules jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    r jsonb;
    w jsonb;
BEGIN
    IF jsonb_typeof(rules) = 'object' THEN
        RETURN public.acl2_rules_valid(rules);
    END IF;
    IF jsonb_typeof(rules) IS DISTINCT FROM 'array' THEN
        RAISE EXCEPTION 'new row for relation "acl_versions" violates check constraint "acl_versions_rules_check"'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'acl_versions_rules_check',
                  SCHEMA = 'public', TABLE = 'acl_versions', COLUMN = 'rules',
                  DETAIL = '{"code": "type", "column": "rules", "pointer": ""}';
    END IF;
    FOR i IN 0 .. jsonb_array_length(rules) - 1 LOOP
        r := rules -> i;
        IF jsonb_typeof(r) <> 'object' THEN
            RETURN false;
        END IF;
        IF r - ARRAY['id', 'ops', 'who', 'where'] <> '{}'::jsonb
           OR jsonb_typeof(r -> 'id') IS DISTINCT FROM 'string' OR r ->> 'id' = ''
           OR jsonb_typeof(r -> 'ops') IS DISTINCT FROM 'array' OR r -> 'ops' = '[]'::jsonb
           OR NOT (r -> 'ops') <@ '["read", "write"]'::jsonb
           OR NOT public.acl_subject_valid(r -> 'who') THEN
            RETURN false;
        END IF;
        w := coalesce(r -> 'where', '[]'::jsonb);
        IF jsonb_typeof(w) <> 'array' THEN
            RETURN false;
        END IF;
        FOR j IN 0 .. jsonb_array_length(w) - 1 LOOP
            IF NOT public.acl_cond_valid(w -> j) THEN
                RETURN false;
            END IF;
        END LOOP;
    END LOOP;
    RETURN true;
END
$$;
-- +goose StatementEnd

-- Every group and user id of a rules or masks value, with its JSON pointer:
-- the strings of a groups or users array (format-2 member_of, format-1 who,
-- mask subjects) and the values of a cmp/Cond on a uuid column. A recursive
-- walk with pointers, so only the trigger's refusal path runs it.
-- +goose StatementBegin
CREATE FUNCTION acl2_id_pointers(doc jsonb) RETURNS TABLE (pointer text, kind text, id text)
    LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
WITH RECURSIVE walk (ptr, key, holder, v) AS (
    SELECT ''::text, NULL::text, NULL::jsonb, doc
    UNION ALL
    SELECT walk.ptr || c.seg, c.key, c.holder, c.v
    FROM walk CROSS JOIN LATERAL (
        SELECT public.acl2_seg(o.key) seg, o.key, walk.v holder, o.value v
        FROM jsonb_each(CASE WHEN jsonb_typeof(walk.v) = 'object' THEN walk.v END) o
        UNION ALL
        SELECT '/' || (a.n - 1), walk.key, walk.holder, a.value
        FROM jsonb_array_elements(CASE WHEN jsonb_typeof(walk.v) = 'array' THEN walk.v END) WITH ORDINALITY a(value, n)
    ) c
)
SELECT ptr,
       CASE WHEN key = 'groups' OR holder ->> 'column' = 'assignment_group_id' THEN 'group' ELSE 'user' END,
       v #>> '{}'
FROM walk
WHERE jsonb_typeof(v) = 'string' AND jsonb_typeof(holder -> key) = 'array'
  AND (key IN ('groups', 'users')
       OR (key = 'values' AND holder ->> 'column' IN ('requester_id', 'assignee_id', 'assignment_group_id')));
$$;
-- +goose StatementEnd

-- The tenant-id backstop (D4, D-11-11): see 5. in the header. One lookup per
-- table over the distinct ids; the pointer walk only when one is foreign.
-- The same answer for an id that does not exist, a tombstoned one and
-- another tenant's (authz.NotInTenant): no oracle for other tenants' ids.
-- +goose StatementBegin
CREATE FUNCTION acl2_tenant_ids() RETURNS trigger
    LANGUAGE plpgsql AS $$
DECLARE
    g uuid[]; u uuid[];
    n_found int;
    col text; ptr text;
BEGIN
    SELECT array_agg(DISTINCT x::uuid) INTO g FROM (
        SELECT jsonb_path_query(NEW.rules, 'lax $.**.groups[*]') #>> '{}' x
        UNION ALL SELECT jsonb_path_query(NEW.rules, 'lax $.** ? (@.column == "assignment_group_id" && exists(@.values)).values[*]') #>> '{}'
        UNION ALL SELECT jsonb_path_query(NEW.masks, 'lax $.**.groups[*]') #>> '{}') s;
    SELECT array_agg(DISTINCT x::uuid) INTO u FROM (
        SELECT jsonb_path_query(NEW.rules, 'lax $.**.users[*]') #>> '{}' x
        UNION ALL SELECT jsonb_path_query(NEW.rules, 'lax $.** ? ((@.column == "requester_id" || @.column == "assignee_id") && exists(@.values)).values[*]') #>> '{}'
        UNION ALL SELECT jsonb_path_query(NEW.masks, 'lax $.**.users[*]') #>> '{}') s;
    IF g IS NOT NULL THEN
        SELECT count(*) INTO n_found FROM public.groups
        WHERE tenant_id = NEW.tenant_id AND deleted_at IS NULL AND id = ANY (g);
        IF n_found = cardinality(g) THEN g := NULL; END IF;
    END IF;
    IF u IS NOT NULL THEN
        SELECT count(*) INTO n_found FROM public.principals
        WHERE tenant_id = NEW.tenant_id AND deleted_at IS NULL AND id = ANY (u);
        IF n_found = cardinality(u) THEN u := NULL; END IF;
    END IF;
    IF g IS NULL AND u IS NULL THEN
        RETURN NULL;
    END IF;
    -- The first foreign id: rules before masks, then by pointer (C
    -- collation), the order the route sorts its problems in.
    SELECT s.col, s.pointer INTO col, ptr FROM (
        SELECT 'rules' col, r.* FROM public.acl2_id_pointers(NEW.rules) r
        UNION ALL SELECT 'masks', m.* FROM public.acl2_id_pointers(NEW.masks) m) s
    WHERE CASE s.kind
          WHEN 'group' THEN NOT EXISTS (SELECT 1 FROM public.groups x
              WHERE x.tenant_id = NEW.tenant_id AND x.deleted_at IS NULL AND x.id = s.id::uuid)
          ELSE NOT EXISTS (SELECT 1 FROM public.principals x
              WHERE x.tenant_id = NEW.tenant_id AND x.deleted_at IS NULL AND x.id = s.id::uuid) END
    ORDER BY s.col = 'masks', s.pointer COLLATE "C"
    LIMIT 1;
    RAISE EXCEPTION 'acl_versions: an id is not a live group or user of the tenant'
        USING ERRCODE = 'check_violation', SCHEMA = 'public', TABLE = 'acl_versions',
              DETAIL = jsonb_build_object('column', coalesce(col, 'rules'), 'code', 'foreign_id',
                                          'pointer', coalesce(ptr, ''))::text;
END
$$;
-- +goose StatementEnd

SET LOCAL lock_timeout = '5s';

ALTER TABLE acl_versions
    ADD CONSTRAINT acl_versions_bytes_rules_check CHECK (NOT (rules ? 'format') OR octet_length(rules::text) <= 131072) NOT VALID,
    ADD CONSTRAINT acl_versions_bytes_masks_check CHECK (octet_length(masks::text) <= 131072) NOT VALID;

CREATE TRIGGER acl_versions_tenant_ids AFTER INSERT ON acl_versions
    FOR EACH ROW EXECUTE FUNCTION acl2_tenant_ids();

-- A1-3 (b): only nava_app publishes.
ALTER POLICY acl_versions_insert ON acl_versions TO nava_app;

-- +goose Down

-- Back to TICK-629: both roles insert, format 1 only, no byte or tenant-id
-- backstop. Safe at any time: it only stops checking new rows.
SET LOCAL lock_timeout = '5s';

ALTER POLICY acl_versions_insert ON acl_versions TO nava_app, nava_worker;
DROP TRIGGER IF EXISTS acl_versions_tenant_ids ON acl_versions;
ALTER TABLE acl_versions
    DROP CONSTRAINT IF EXISTS acl_versions_bytes_rules_check,
    DROP CONSTRAINT IF EXISTS acl_versions_bytes_masks_check;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl_rules_valid(rules jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    r jsonb;
    w jsonb;
BEGIN
    IF jsonb_typeof(rules) IS DISTINCT FROM 'array' THEN
        RETURN false;
    END IF;
    FOR i IN 0 .. jsonb_array_length(rules) - 1 LOOP
        r := rules -> i;
        IF jsonb_typeof(r) <> 'object' THEN
            RETURN false;
        END IF;
        IF r - ARRAY['id', 'ops', 'who', 'where'] <> '{}'::jsonb
           OR jsonb_typeof(r -> 'id') IS DISTINCT FROM 'string' OR r ->> 'id' = ''
           OR jsonb_typeof(r -> 'ops') IS DISTINCT FROM 'array' OR r -> 'ops' = '[]'::jsonb
           OR NOT (r -> 'ops') <@ '["read", "write"]'::jsonb
           OR NOT public.acl_subject_valid(r -> 'who') THEN
            RETURN false;
        END IF;
        w := coalesce(r -> 'where', '[]'::jsonb);
        IF jsonb_typeof(w) <> 'array' THEN
            RETURN false;
        END IF;
        FOR j IN 0 .. jsonb_array_length(w) - 1 LOOP
            IF NOT public.acl_cond_valid(w -> j) THEN
                RETURN false;
            END IF;
        END LOOP;
    END LOOP;
    RETURN true;
END
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl_uuids_valid(a jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    e jsonb;
    t text;
BEGIN
    IF jsonb_typeof(a) IS DISTINCT FROM 'array' THEN
        RETURN false;
    END IF;
    FOR i IN 0 .. jsonb_array_length(a) - 1 LOOP
        e := a -> i;
        IF jsonb_typeof(e) <> 'string' THEN
            RETURN false;
        END IF;
        t := e #>> '{}';
        IF length(t) <> 36 OR t !~ '^[0-9A-Fa-f-]+$' OR length(replace(t, '-', '')) <> 32
           OR substr(t, 9, 1) <> '-' OR substr(t, 14, 1) <> '-' OR substr(t, 19, 1) <> '-' OR substr(t, 24, 1) <> '-' THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
END
$$;
-- +goose StatementEnd

DROP FUNCTION IF EXISTS acl2_tenant_ids();
DROP FUNCTION IF EXISTS acl2_id_pointers(jsonb);
DROP FUNCTION IF EXISTS acl2_rules_valid(jsonb);
DROP FUNCTION IF EXISTS acl2_problem(jsonb);
DROP FUNCTION IF EXISTS acl2_expr(jsonb, text);
DROP FUNCTION IF EXISTS acl2_under_problem(jsonb, text);
DROP FUNCTION IF EXISTS acl2_cond_problem(jsonb, text);
DROP FUNCTION IF EXISTS acl2_member_of_problem(jsonb, text);
DROP FUNCTION IF EXISTS acl2_ids_problem(jsonb, text);
DROP FUNCTION IF EXISTS acl2_name_ok(text);
DROP FUNCTION IF EXISTS acl2_unknown_key(jsonb, text, text[]);
DROP FUNCTION IF EXISTS acl2_seg(text);
DROP FUNCTION IF EXISTS acl2_bad(text, text);
