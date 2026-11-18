-- SP-2 (TICK-933) PROTOTYPE, not a migration. The format-2 shape CHECK and
-- the tenant-id trigger for acl_versions (ADR-010 D2-D4, PLAN-011 D-11-5,
-- D-11-11), written to measure their insert cost at the D4 caps. TICK-940
-- P11-08 turns it into migrations; the numbers are in
-- docs/data/acl-format2-caps.md.
--
-- Needs the TICK-629 functions (acl_uuids_valid, acl_cond_valid,
-- acl_subject_valid, acl_rules_valid, acl_masks_valid) from
-- db/migrations/20261109100000_acl_versions_shape_check.sql.
--
-- Format 2 (D2, D3, D-11-5):
--   {"format": 2, "conditions": [{name, expr}], "policies": [{name, ops, when}]}
--   name   ^[a-z][a-z0-9_]{0,62}$, unique within its array
--   ops    non-empty array, each "read" or "write"
--   expr   an object with exactly one key:
--          all / any   array of 1 to 16 expr
--          not         expr
--          ref         the name of a condition of this document
--          member_of   {groups?, users?}: non-empty uuid arrays, at least one
--          cmp         a format-1 Cond (acl_cond_valid)
--          shared      true
--          under       {"of": "caller"} or {"of": [uuid, ...]} (non-empty)
--
-- Counting (D4). A leaf is member_of, cmp, shared or under. A ref is
-- replaced by its condition's expr: it adds that expr's leaves, and its depth
-- is the ref's own depth plus that expr's depth minus one. Depth of a lone
-- leaf is 1; all/any/not add one. Leaves and depth are memoized per
-- condition (Kahn's order over the reference graph), never expanded, so a
-- doubling chain costs its written size, not 2^n. A condition still
-- unresolved when the queue empties is on a cycle (self-reference
-- included). Caps apply to policies, because only they are expanded (D4: an
-- unused condition is allowed); refs and cycles are checked for every
-- condition.
--
-- The cap parameters default to the D4 values, so the bench can measure a
-- document at the D4 caps. TICK-940 ships SP-2's caps instead: 8, 16, 64,
-- 128 and 128 (docs/data/acl-format2-caps.md, item 1).
--
-- acl2_problem returns NULL for a valid document or the C1 code of the first
-- problem found; acl2_rules_valid is the CHECK predicate. A format-1 value
-- (no "format" key) goes to the TICK-629 function unchanged.

-- A replacement BODY for TICK-629's acl_uuids_valid, same contract (an
-- array of 8-4-4-4-12 hex strings, either case; [] is valid): one regular
-- expression over the array's canonical jsonb text instead of a loop per
-- element. jsonb prints an array as ["a", "b"] with ", " between elements,
-- and a uuid string has nothing to escape, so the text form is exact; a
-- non-string, nested or malformed element breaks the pattern. No counted
-- repetition ({8}), which costs 6.6 us a match in PostgreSQL's engine. At
-- 5,000 ids: 5 ms against 141 ms for the loop (docs/data/acl-format2-caps.md).
-- Format 1 gets it too, through acl_subject_valid and acl_cond_valid.
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

-- One expression: shape, and the counts with refs left symbolic. A worklist,
-- not recursion: leaves and depth are a count and a max, so visiting order
-- does not matter, and a PL/pgSQL call per node costs more than an array
-- slot.
CREATE OR REPLACE FUNCTION acl2_expr(e jsonb,
    OUT problem text, OUT leaves bigint, OUT depth int, OUT refs text[], OUT ref_depths int[])
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    st  jsonb[] := ARRAY[e];
    sd  int[]   := ARRAY[1];
    top int     := 1;
    n   jsonb;
    d   int;
    k   text;
    v   jsonb;
    len int;
    nr  int := 0;
BEGIN
    leaves := 0; depth := 0; refs := '{}'; ref_depths := '{}';
    WHILE top > 0 LOOP
        n := st[top]; d := sd[top]; top := top - 1;
        IF jsonb_typeof(n) IS DISTINCT FROM 'object' THEN
            problem := 'unknown_key'; RETURN;
        END IF;
        -- Exactly one key, from the closed set. Plain expressions, not a
        -- SELECT over jsonb_object_keys: a query per node is an SPI plan
        -- execution, and that was most of the cost (docs/data/acl-format2-caps.md).
        k := CASE WHEN n ? 'cmp' THEN 'cmp' WHEN n ? 'all' THEN 'all' WHEN n ? 'any' THEN 'any'
                  WHEN n ? 'ref' THEN 'ref' WHEN n ? 'not' THEN 'not' WHEN n ? 'member_of' THEN 'member_of'
                  WHEN n ? 'shared' THEN 'shared' WHEN n ? 'under' THEN 'under' END;
        IF k IS NULL OR n - k <> '{}'::jsonb THEN
            problem := 'unknown_key'; RETURN;
        END IF;
        v := n -> k;
        CASE k
        WHEN 'all', 'any' THEN
            IF jsonb_typeof(v) <> 'array' THEN problem := 'unknown_key'; RETURN; END IF;
            len := jsonb_array_length(v);
            IF len < 1 OR len > 16 THEN problem := 'child_count'; RETURN; END IF;
            FOR i IN 0 .. len - 1 LOOP
                top := top + 1; st[top] := v -> i; sd[top] := d + 1;
            END LOOP;
        WHEN 'not' THEN
            top := top + 1; st[top] := v; sd[top] := d + 1;
        WHEN 'ref' THEN
            IF jsonb_typeof(v) <> 'string' THEN problem := 'unresolved_ref'; RETURN; END IF;
            nr := nr + 1; refs[nr] := v #>> '{}'; ref_depths[nr] := d;
        WHEN 'member_of' THEN
            IF jsonb_typeof(v) <> 'object' OR v = '{}'::jsonb
               OR v - ARRAY['groups', 'users'] <> '{}'::jsonb
               OR (v ? 'groups' AND (v -> 'groups' = '[]'::jsonb OR NOT public.acl_uuids_valid(v -> 'groups')))
               OR (v ? 'users' AND (v -> 'users' = '[]'::jsonb OR NOT public.acl_uuids_valid(v -> 'users'))) THEN
                problem := 'unknown_key'; RETURN;
            END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'cmp' THEN
            IF NOT public.acl_cond_valid(v) THEN problem := 'unknown_key'; RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'shared' THEN
            IF v IS DISTINCT FROM 'true'::jsonb THEN problem := 'unknown_key'; RETURN; END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        WHEN 'under' THEN
            IF jsonb_typeof(v) <> 'object' OR v - 'of' <> '{}'::jsonb
               OR NOT (v -> 'of' = '"caller"'::jsonb
                       OR (jsonb_typeof(v -> 'of') = 'array' AND v -> 'of' <> '[]'::jsonb
                           AND public.acl_uuids_valid(v -> 'of'))) THEN
                problem := 'unknown_key'; RETURN;
            END IF;
            leaves := leaves + 1; depth := greatest(depth, d);
        ELSE
            problem := 'unknown_key'; RETURN;
        END CASE;
    END LOOP;
END
$$;

CREATE OR REPLACE FUNCTION acl2_problem(doc jsonb, max_depth int DEFAULT 8, max_policy_leaves int DEFAULT 256,
    max_policies int DEFAULT 64, max_doc_leaves int DEFAULT 1024) RETURNS text
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    -- Saturate leaf counts here: a doubling chain overflows bigint at 63 levels.
    sat CONSTANT bigint := 1000000000;
    conds jsonb; pols jsonb; c jsonb; p jsonb;
    nc int; np int;
    idx jsonb := '{}';            -- condition name -> 1-based index
    pidx jsonb := '{}';           -- policy name -> true
    w record;
    lv bigint[]; dp int[];        -- per condition: local counts, then totals
    outdeg int[];
    -- edges i -> j (condition i refs condition j at depth ed)
    efrom int[] := '{}'; eto int[] := '{}'; ed int[] := '{}'; ne int := 0;
    -- reverse adjacency (edges grouped by eto): rstart[j] .. rstart[j+1]-1 in rorder
    rstart int[]; rorder int[]; fill int[];
    q int[]; qh int := 1; qt int := 0;
    acc_l bigint[]; acc_d int[];
    j int; f int; e int;
    pl bigint; pd int; tot bigint := 0;
    nm text;
    ok_shape boolean; ok_names boolean; ok_unique boolean;
BEGIN
    IF jsonb_typeof(doc) IS DISTINCT FROM 'object' THEN
        RETURN 'unknown_key';
    END IF;
    IF doc - ARRAY['format', 'conditions', 'policies'] <> '{}'::jsonb OR doc -> 'format' IS DISTINCT FROM '2'::jsonb
       OR jsonb_typeof(doc -> 'conditions') IS DISTINCT FROM 'array'
       OR jsonb_typeof(doc -> 'policies') IS DISTINCT FROM 'array' THEN
        RETURN 'unknown_key';
    END IF;
    conds := doc -> 'conditions'; pols := doc -> 'policies';
    nc := jsonb_array_length(conds); np := jsonb_array_length(pols);
    IF np > max_policies THEN
        RETURN 'policies_exceeded';
    END IF;

    -- Condition names, in one query: name -> 1-based index. Growing a jsonb
    -- object one name at a time copies it each time (quadratic in the
    -- number of conditions, which no cap bounds).
    SELECT bool_and(jsonb_typeof(ce) = 'object' AND ce - ARRAY['name', 'expr'] = '{}'::jsonb AND ce ? 'expr'
                    AND jsonb_typeof(ce -> 'name') = 'string'),
           bool_and(ce ->> 'name' ~ '^[a-z][a-z0-9_]*$' AND length(ce ->> 'name') <= 63),
           count(DISTINCT ce ->> 'name') = count(*),
           jsonb_object_agg(coalesce(ce ->> 'name', ''), co)
      INTO ok_shape, ok_names, ok_unique, idx
      FROM jsonb_array_elements(conds) WITH ORDINALITY AS t(ce, co);
    IF nc > 0 THEN
        IF NOT ok_shape THEN RETURN 'unknown_key'; END IF;
        IF NOT ok_names THEN RETURN 'invalid_name'; END IF;
        IF NOT ok_unique THEN RETURN 'duplicate_name'; END IF;
    ELSE
        idx := '{}';
    END IF;

    -- Each condition once: shape, local counts, its edges.
    lv := array_fill(0::bigint, ARRAY[nc]); dp := array_fill(0, ARRAY[nc]); outdeg := array_fill(0, ARRAY[nc]);
    FOR i IN 1 .. nc LOOP
        w := public.acl2_expr(conds -> (i - 1) -> 'expr');
        IF w.problem IS NOT NULL THEN RETURN w.problem; END IF;
        lv[i] := w.leaves; dp[i] := w.depth;
        FOR k IN 1 .. coalesce(array_length(w.refs, 1), 0) LOOP
            IF NOT idx ? w.refs[k] THEN RETURN 'unresolved_ref'; END IF;
            ne := ne + 1; efrom[ne] := i; eto[ne] := (idx ->> w.refs[k])::int; ed[ne] := w.ref_depths[k];
            outdeg[i] := outdeg[i] + 1;
        END LOOP;
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
        IF qt < nc THEN RETURN 'ref_cycle'; END IF;
    END IF;

    -- Policies: expanded counts against the caps.
    FOR i IN 1 .. np LOOP
        p := pols -> (i - 1);
        IF jsonb_typeof(p) <> 'object' OR p - ARRAY['name', 'ops', 'when'] <> '{}'::jsonb OR NOT p ? 'when'
           OR jsonb_typeof(p -> 'name') IS DISTINCT FROM 'string'
           OR jsonb_typeof(p -> 'ops') IS DISTINCT FROM 'array' OR p -> 'ops' = '[]'::jsonb
           OR NOT (p -> 'ops') <@ '["read", "write"]'::jsonb THEN
            RETURN 'unknown_key';
        END IF;
        nm := p ->> 'name';
        IF nm !~ '^[a-z][a-z0-9_]*$' OR length(nm) > 63 THEN RETURN 'invalid_name'; END IF;
        IF pidx ? nm THEN RETURN 'duplicate_name'; END IF;
        pidx := pidx || jsonb_build_object(nm, true);
        w := public.acl2_expr(p -> 'when');
        IF w.problem IS NOT NULL THEN RETURN w.problem; END IF;
        pl := w.leaves; pd := w.depth;
        FOR k IN 1 .. coalesce(array_length(w.refs, 1), 0) LOOP
            IF NOT idx ? w.refs[k] THEN RETURN 'unresolved_ref'; END IF;
            j := (idx ->> w.refs[k])::int;
            pl := least(sat, pl + lv[j]); pd := greatest(pd, w.ref_depths[k] + dp[j] - 1);
        END LOOP;
        IF pd > max_depth THEN RETURN 'depth_exceeded'; END IF;
        IF pl > max_policy_leaves THEN RETURN 'policy_leaves_exceeded'; END IF;
        tot := tot + pl;
        IF tot > max_doc_leaves THEN RETURN 'document_leaves_exceeded'; END IF;
    END LOOP;
    RETURN NULL;
END
$$;

-- The CHECK predicate: format 2 by its key, format 1 unchanged (stored rows
-- stay valid; the route never stores format 1, PLAN-011 D12 note).
CREATE OR REPLACE FUNCTION acl2_rules_valid(rules jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    IF jsonb_typeof(rules) = 'object' AND rules ? 'format' THEN
        RETURN public.acl2_problem(rules) IS NULL;
    END IF;
    RETURN public.acl_rules_valid(rules);
END
$$;

-- Tenant ids (D4, D-11-11). Every group, user and org-unit id in rules and
-- masks must be a LIVE row of NEW.tenant_id; the lookup filters tenant_id
-- explicitly, on top of RLS. One query per table over the distinct ids.
--
-- Written as an AFTER INSERT row trigger, not BEFORE: an AFTER trigger fires
-- after the CHECK, so it only sees a shape-valid document and every id it
-- casts is a uuid string. A BEFORE trigger sees the raw value and must run
-- the shape check itself first (twice the CHECK cost) or turn a bad id into
-- 22P02 instead of 23514. Raising in an AFTER (non-deferred) trigger aborts
-- the statement just the same.
--
-- Paths: lax $.** reaches the format-2 member_of / cmp / under nodes, the
-- format-1 who / where objects and the mask subjects with one expression per
-- kind. A uuid column of a cmp is a group column (assignment_group_id) or a
-- user column (requester_id, assignee_id), as policycheck.go does.
CREATE OR REPLACE FUNCTION acl2_tenant_ids() RETURNS trigger
    LANGUAGE plpgsql AS $$
DECLARE
    g uuid[]; u uuid[]; o uuid[];
    found int;
BEGIN
    SELECT array_agg(DISTINCT x::text::uuid) INTO g FROM (
        SELECT jsonb_path_query(NEW.rules, 'lax $.**.groups[*]') #>> '{}' x
        UNION ALL SELECT jsonb_path_query(NEW.rules, 'lax $.** ? (@.column == "assignment_group_id" && exists(@.values)).values[*]') #>> '{}'
        UNION ALL SELECT jsonb_path_query(NEW.masks, 'lax $.**.groups[*]') #>> '{}') s;
    SELECT array_agg(DISTINCT x::text::uuid) INTO u FROM (
        SELECT jsonb_path_query(NEW.rules, 'lax $.**.users[*]') #>> '{}' x
        UNION ALL SELECT jsonb_path_query(NEW.rules, 'lax $.** ? ((@.column == "requester_id" || @.column == "assignee_id") && exists(@.values)).values[*]') #>> '{}'
        UNION ALL SELECT jsonb_path_query(NEW.masks, 'lax $.**.users[*]') #>> '{}') s;
    SELECT array_agg(DISTINCT x::text::uuid) INTO o FROM (
        SELECT jsonb_path_query(NEW.rules, 'lax $.**.under.of ? (@.type() == "array")[*]') #>> '{}' x) s;
    IF g IS NOT NULL THEN
        SELECT count(*) INTO found FROM public.groups
        WHERE tenant_id = NEW.tenant_id AND deleted_at IS NULL AND id = ANY(g);
        IF found <> cardinality(g) THEN
            RAISE EXCEPTION 'acl_versions: a group id is not a live group of the tenant'
                USING ERRCODE = 'check_violation', DETAIL = 'foreign_id';
        END IF;
    END IF;
    IF u IS NOT NULL THEN
        SELECT count(*) INTO found FROM public.principals
        WHERE tenant_id = NEW.tenant_id AND deleted_at IS NULL AND id = ANY(u);
        IF found <> cardinality(u) THEN
            RAISE EXCEPTION 'acl_versions: a user id is not a live principal of the tenant'
                USING ERRCODE = 'check_violation', DETAIL = 'foreign_id';
        END IF;
    END IF;
    IF o IS NOT NULL THEN
        SELECT count(*) INTO found FROM public.org_units
        WHERE tenant_id = NEW.tenant_id AND deleted_at IS NULL AND id = ANY(o);
        IF found <> cardinality(o) THEN
            RAISE EXCEPTION 'acl_versions: an org-unit id is not a live org unit of the tenant'
                USING ERRCODE = 'check_violation', DETAIL = 'foreign_id';
        END IF;
    END IF;
    RETURN NULL;
END
$$;
