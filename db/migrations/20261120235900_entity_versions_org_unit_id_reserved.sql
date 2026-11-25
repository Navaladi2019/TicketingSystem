-- TICK-1115 (S31, ADR-010 D8): no stored entity-type version may have a
-- top-level property named org_unit_id. S31 reserves the name, because the
-- pipeline puts records.org_unit_id in the audit payload under it. Compile
-- refuses it and also loads stored versions, and releases before S31 accepted
-- it, so without this an upgrade succeeds and every read and write that loads
-- such a version then fails.
--
-- The upgrade stops here instead, with 23514 and this constraint's name. The
-- API cannot repair a refused row: publish compiles the previous version
-- first and fails the same way, and versions are immutable (D-3-4). The
-- constraint also refuses a publish by a replica of the previous release
-- between this migration and the rollout. Compile stays the check the API
-- answers from. It scans every stored row whatever RLS says, so it needs no
-- temporary policy.
--
-- It must stay S31's first migration. A refused run stops before the grants
-- manifest is applied, and every migration before the refusal stays committed.
-- Stamped last, it left 11 S31 migrations applied without their grants, and
-- every tenant's access publish on the running release then failed with 42501
-- (permission denied for function acl2_max_under_sets) until a superuser
-- repaired one tenant's row. If main gets a newer migration before S31 ships,
-- re-stamp every S31 migration and keep this one first
-- (TestTICK1115RefusalAppliesNothingOfS31).
--
-- Top level only, as in Compile: a nested property of that name is legal.
--
-- The expression replaces escapes before it reads the key, because json ->
-- de-escapes the whole document and raises (22P05, 22P02) on a \u0000 escape
-- or a lone surrogate, which the json column stores. Such a row must not block
-- the upgrade. Escaped backslashes go first, so what follows a remaining
-- backslash is an escape. No key looked up holds a "?", so a replaced key
-- never becomes one of them, and an escaped spelling of a key (\u006f for o)
-- is still found. Never cast the column to jsonb: that raises too.
--
-- Lock: ACCESS EXCLUSIVE on entity_definition_versions while it scans, a small
-- table with one row per published version.
-- Figures: docs/data/entity-versions-org-unit-id-reserved.md.
--
-- The refused versions, for an operator connected as a superuser (RLS hides
-- them from every other role):
--
--   SELECT v.tenant_id, d.key, v.version, v.version = d.current_version AS is_current
--   FROM entity_definition_versions v
--   JOIN entity_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
--   WHERE (replace(replace(replace(replace(v.schema::text,
--       '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json
--       -> 'properties' -> 'org_unit_id') IS NOT NULL
--   ORDER BY 1, 2, 3;

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions
    ADD CONSTRAINT entity_definition_versions_org_unit_id_reserved
    CHECK ((replace(replace(replace(replace(schema::text,
        '\\', '?'), '\u0000', '?'), '\ud', '?'), '\uD', '?')::json
        -> 'properties' -> 'org_unit_id') IS NULL);

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions
    DROP CONSTRAINT entity_definition_versions_org_unit_id_reserved;
