-- TICK-629 (deferred from P3-31): acl_versions.rules and masks must have
-- the shape internal/authz decodes and the S10 contract (amendment A1)
-- defines. PolicyStore.Current decodes them strictly (unknown keys refused),
-- so one malformed row turns every read of its entity type into a 500, and
-- versions are immutable: no role may UPDATE a bad row afterwards. The
-- insert is the only place to stop it.
--
-- A pure CHECK, not a trigger: the shape is a property of the one value, so
-- an IMMUTABLE function over the jsonb expresses all of it. PL/pgSQL, not
-- LANGUAGE sql: the SQL-language version could not be inlined (it walks
-- arrays with subqueries) and re-planned every nested call: 1.3 ms a
-- 6-rule policy against 0.35 ms here (docs/data/acl-versions-shape-check.md).
-- Not SECURITY DEFINER, so they run as the inserting role; calls between
-- them are schema-qualified, and the operators and built-ins they use
-- resolve in pg_catalog, which is searched first.
--
-- The accepted shape (docs/data/acl-versions-shape-check.md; the publish
-- validation P3-35 mirrors it). No value anywhere may be JSON null: a key is
-- either absent or has its type. No key outside those listed.
--
--   rules  array of Rule (may be empty: the policy allows nothing)
--   Rule   {id, ops, who, where?}
--          id     non-empty string
--          ops    non-empty array, each "read" or "write"
--          who    Subject
--          where  array of Cond (may be empty: every record)
--   Subject {groups?, users?, requester?, assignee?, shared?}
--          groups, users                  arrays of uuid strings (may be empty)
--          requester, assignee, shared    booleans
--   Cond   {column, op, values?, caller?}
--          column  state | requester_id | assignee_id | assignment_group_id
--          op      eq | in
--          caller  "" (or absent): values are literals. Every value is a
--                    string, a uuid string unless column is state; eq takes
--                    exactly one value, in any number (none matches nothing).
--                  self:      column requester_id or assignee_id, no values
--                  my_groups: column assignment_group_id, no values
--   masks  array of Mask (may be empty)
--   Mask   {field, read, write}
--          field        non-empty string
--          read, write  Subject
--
-- A uuid string is the 8-4-4-4-12 hex form, either case: the form
-- encoding/json writes for uuid.UUID, and one uuid.Parse accepts.
--
-- The Cond rules beyond the decode (eq takes one value; self and my_groups
-- only on their columns and with no values) are the cases the compiler drops
-- as "matches nothing": a policy that says them is a mistake, so it is
-- refused here rather than silently denying.
--
-- nava_app and nava_worker insert acl_versions, and a CHECK runs its
-- functions as the inserting role, so both get EXECUTE on all five
-- (db/grants.yaml). PUBLIC has none (00001's default privileges).
--
-- Added NOT VALID here, under a short ACCESS EXCLUSIVE lock (catalogue only,
-- no scan), and validated by the next migration under SHARE UPDATE
-- EXCLUSIVE, which blocks neither reads nor publishes. New rows are checked
-- from this migration on. Figures: docs/data/acl-versions-shape-check.md.

-- +goose Up

-- An array of uuid strings: 36 characters, hex digits and hyphens only,
-- exactly four hyphens, at 9, 14, 19 and 24. The same set as
-- ^[0-9A-Fa-f]{8}-([0-9A-Fa-f]{4}-){3}[0-9A-Fa-f]{12}$ at a quarter of its
-- cost: PostgreSQL's counted repetition takes 6.6 us a match, and a policy
-- holds dozens of ids.
-- +goose StatementBegin
CREATE FUNCTION acl_uuids_valid(a jsonb) RETURNS boolean
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

-- authz.Subject.
-- +goose StatementBegin
CREATE FUNCTION acl_subject_valid(s jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
BEGIN
    IF jsonb_typeof(s) IS DISTINCT FROM 'object' THEN
        RETURN false;
    END IF;
    IF s - ARRAY['groups', 'users', 'requester', 'assignee', 'shared'] <> '{}'::jsonb THEN
        RETURN false;
    END IF;
    IF (s ? 'groups' AND NOT public.acl_uuids_valid(s -> 'groups'))
       OR (s ? 'users' AND NOT public.acl_uuids_valid(s -> 'users')) THEN
        RETURN false;
    END IF;
    RETURN coalesce(jsonb_typeof(s -> 'requester'), 'boolean') = 'boolean'
       AND coalesce(jsonb_typeof(s -> 'assignee'), 'boolean') = 'boolean'
       AND coalesce(jsonb_typeof(s -> 'shared'), 'boolean') = 'boolean';
END
$$;
-- +goose StatementEnd

-- authz.Cond.
-- +goose StatementBegin
CREATE FUNCTION acl_cond_valid(c jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    col    text;
    op     text;
    caller text;
    vals   jsonb;
BEGIN
    IF jsonb_typeof(c) IS DISTINCT FROM 'object' THEN
        RETURN false;
    END IF;
    IF c - ARRAY['column', 'op', 'values', 'caller'] <> '{}'::jsonb
       OR jsonb_typeof(c -> 'column') IS DISTINCT FROM 'string'
       OR jsonb_typeof(c -> 'op') IS DISTINCT FROM 'string'
       OR (c ? 'values' AND jsonb_typeof(c -> 'values') <> 'array')
       OR (c ? 'caller' AND jsonb_typeof(c -> 'caller') <> 'string') THEN
        RETURN false;
    END IF;
    col := c ->> 'column';
    op := c ->> 'op';
    caller := coalesce(c ->> 'caller', '');
    vals := coalesce(c -> 'values', '[]'::jsonb);
    IF col NOT IN ('state', 'requester_id', 'assignee_id', 'assignment_group_id') OR op NOT IN ('eq', 'in') THEN
        RETURN false;
    END IF;
    IF caller = '' THEN
        IF op = 'eq' AND jsonb_array_length(vals) <> 1 THEN
            RETURN false;
        END IF;
        IF col <> 'state' THEN
            RETURN public.acl_uuids_valid(vals);
        END IF;
        FOR i IN 0 .. jsonb_array_length(vals) - 1 LOOP
            IF jsonb_typeof(vals -> i) <> 'string' THEN
                RETURN false;
            END IF;
        END LOOP;
        RETURN true;
    END IF;
    IF vals <> '[]'::jsonb THEN
        RETURN false;
    END IF;
    RETURN (caller = 'self' AND col IN ('requester_id', 'assignee_id'))
        OR (caller = 'my_groups' AND col = 'assignment_group_id');
END
$$;
-- +goose StatementEnd

-- acl_versions.rules: []authz.Rule.
-- +goose StatementBegin
CREATE FUNCTION acl_rules_valid(rules jsonb) RETURNS boolean
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

-- acl_versions.masks: []authz.Mask.
-- +goose StatementBegin
CREATE FUNCTION acl_masks_valid(masks jsonb) RETURNS boolean
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE
    m jsonb;
BEGIN
    IF jsonb_typeof(masks) IS DISTINCT FROM 'array' THEN
        RETURN false;
    END IF;
    FOR i IN 0 .. jsonb_array_length(masks) - 1 LOOP
        m := masks -> i;
        IF jsonb_typeof(m) <> 'object' THEN
            RETURN false;
        END IF;
        IF m - ARRAY['field', 'read', 'write'] <> '{}'::jsonb
           OR jsonb_typeof(m -> 'field') IS DISTINCT FROM 'string' OR m ->> 'field' = ''
           OR NOT public.acl_subject_valid(m -> 'read')
           OR NOT public.acl_subject_valid(m -> 'write') THEN
            RETURN false;
        END IF;
    END LOOP;
    RETURN true;
END
$$;
-- +goose StatementEnd

SET LOCAL lock_timeout = '5s';

ALTER TABLE acl_versions
    ADD CONSTRAINT acl_versions_rules_check CHECK (acl_rules_valid(rules)) NOT VALID,
    ADD CONSTRAINT acl_versions_masks_check CHECK (acl_masks_valid(masks)) NOT VALID;

-- +goose Down

-- Drops the CHECKs, then the functions, callers first. Safe at any time: it
-- only stops checking new rows.
SET LOCAL lock_timeout = '5s';

ALTER TABLE acl_versions
    DROP CONSTRAINT IF EXISTS acl_versions_rules_check,
    DROP CONSTRAINT IF EXISTS acl_versions_masks_check;
DROP FUNCTION IF EXISTS acl_masks_valid(jsonb);
DROP FUNCTION IF EXISTS acl_rules_valid(jsonb);
DROP FUNCTION IF EXISTS acl_cond_valid(jsonb);
DROP FUNCTION IF EXISTS acl_subject_valid(jsonb);
DROP FUNCTION IF EXISTS acl_uuids_valid(jsonb);
