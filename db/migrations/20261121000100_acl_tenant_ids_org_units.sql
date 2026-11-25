-- TICK-947 (S31 P11-14; ADR-010 D4, D7; PLAN-011 D-11-11): the acl_versions
-- tenant-id trigger (TICK-940, 20261119100000) checks org-unit ids too.
--
-- 1. Every id in an under {"of": [ids]} of rules must be a live org unit of
--    NEW.tenant_id, filtered explicitly, not only by RLS (the owner and the
--    superuser bypass it). A tombstoned unit, another tenant's unit and an
--    id of no unit get the same answer, 23514 with DETAIL {"column":
--    "rules", "code": "foreign_id", "pointer"}, as groups and users do: no
--    oracle for which ids exist (TICK-940 security review, Technical Lead on
--    TICK-947, 2026-11-19). under {"of": "caller"} names no id. masks have
--    no under. One lookup over the distinct ids, backed by org_units_pkey
--    (tenant_id, id); the pointer walk runs only on a refusal.
--
-- 2. acl2_id_pointers reports those ids as kind 'org_unit'. "of" is a key
--    of under only (acl2_under_problem refuses any other key there, and no
--    other node has one), so the walk keys on it.
--
-- 3. AR-S30-01: before the trigger checks org-unit ids, a stored row could
--    hold one the route never vetted. This migration refuses to apply if any
--    stored acl_versions row names an under id that is not a live org unit
--    of the row's own tenant. acl_versions and org_units are FORCE RLS and
--    nava_migrator has no policy on them, so the check reads through a
--    SELECT USING (true) policy for nava_migrator that exists only inside
--    this transaction (the P3-40 / TICK-715 pattern); both are dropped before
--    COMMIT, and a failure rolls them back with everything else.
--
-- Still SECURITY INVOKER (I3): it runs as nava_app, which reads org_units.
--
-- Locks: CREATE OR REPLACE FUNCTION takes no table lock. The two temporary
-- policies take ACCESS EXCLUSIVE on acl_versions and org_units to COMMIT,
-- both small (one row per published access version; one per org unit).

-- +goose Up

SET LOCAL lock_timeout = '5s';

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl2_id_pointers(doc jsonb) RETURNS TABLE (pointer text, kind text, id text)
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
       CASE WHEN key = 'of' THEN 'org_unit'
            WHEN key = 'groups' OR holder ->> 'column' = 'assignment_group_id' THEN 'group'
            ELSE 'user' END,
       v #>> '{}'
FROM walk
WHERE jsonb_typeof(v) = 'string' AND jsonb_typeof(holder -> key) = 'array'
  AND (key IN ('groups', 'users', 'of')
       OR (key = 'values' AND holder ->> 'column' IN ('requester_id', 'assignee_id', 'assignment_group_id')));
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl2_tenant_ids() RETURNS trigger
    LANGUAGE plpgsql AS $$
DECLARE
    g uuid[]; u uuid[]; o uuid[];
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
    -- of is an array of ids or the string "caller"; type() does not unwrap.
    SELECT array_agg(DISTINCT x::uuid) INTO o FROM (
        SELECT jsonb_path_query(NEW.rules, 'lax $.**.under ? (@.of.type() == "array").of[*]') #>> '{}' x) s;
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
    IF o IS NOT NULL THEN
        SELECT count(*) INTO n_found FROM public.org_units
        WHERE tenant_id = NEW.tenant_id AND deleted_at IS NULL AND id = ANY (o);
        IF n_found = cardinality(o) THEN o := NULL; END IF;
    END IF;
    IF g IS NULL AND u IS NULL AND o IS NULL THEN
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
          WHEN 'org_unit' THEN NOT EXISTS (SELECT 1 FROM public.org_units x
              WHERE x.tenant_id = NEW.tenant_id AND x.deleted_at IS NULL AND x.id = s.id::uuid)
          ELSE NOT EXISTS (SELECT 1 FROM public.principals x
              WHERE x.tenant_id = NEW.tenant_id AND x.deleted_at IS NULL AND x.id = s.id::uuid) END
    ORDER BY s.col = 'masks', s.pointer COLLATE "C"
    LIMIT 1;
    RAISE EXCEPTION 'acl_versions: an id is not a live group or user (or org unit) of the tenant'
        USING ERRCODE = 'check_violation', SCHEMA = 'public', TABLE = 'acl_versions',
              DETAIL = jsonb_build_object('column', coalesce(col, 'rules'), 'code', 'foreign_id',
                                          'pointer', coalesce(ptr, ''))::text;
END
$$;
-- +goose StatementEnd

-- AR-S30-01: no stored row names an org unit outside its own tenant.
CREATE POLICY acl_versions_migrator_validate ON acl_versions FOR SELECT TO nava_migrator USING (true);
CREATE POLICY org_units_migrator_validate ON org_units FOR SELECT TO nava_migrator USING (true);
-- +goose StatementBegin
DO $$
DECLARE
    bad bigint;
BEGIN
    SELECT count(*) INTO bad
    FROM public.acl_versions a
    CROSS JOIN LATERAL jsonb_path_query(a.rules, 'lax $.**.under ? (@.of.type() == "array").of[*]') x
    WHERE NOT EXISTS (SELECT 1 FROM public.org_units o
                      WHERE o.tenant_id = a.tenant_id AND o.deleted_at IS NULL AND o.id = (x #>> '{}')::uuid);
    IF bad > 0 THEN
        RAISE EXCEPTION 'AR-S30-01: % stored under id(s) in acl_versions are not a live org unit of the row''s tenant', bad
            USING ERRCODE = 'check_violation',
                  HINT = 'Do not delete acl_versions rows: the newest row is the access in force. List the ids with the AR-S30-01 query in docs/data/entity-versions-org-unit-id-reserved.md and ask the platform operator''s support before migrating again.';
    END IF;
END
$$;
-- +goose StatementEnd
DROP POLICY org_units_migrator_validate ON org_units;
DROP POLICY acl_versions_migrator_validate ON acl_versions;

-- +goose Down

-- Back to TICK-940's functions: org-unit ids are no longer checked.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl2_id_pointers(doc jsonb) RETURNS TABLE (pointer text, kind text, id text)
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION acl2_tenant_ids() RETURNS trigger
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
