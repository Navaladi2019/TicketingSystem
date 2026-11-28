-- P7-12a (TICK-1172): a version after v1 may name a system publisher instead
-- of a principal. S22's onboarding (TICK-752) publishes version 2 of each seed
-- ticket type, with its filterable fields flagged, while the tenant is
-- provisioning and has no principal. entity_definition_versions_published_by_check
-- (20261101130000, P3-06c) refuses that row. The Technical Lead chose to record
-- the system publisher in its own column (option 1 on TICK-752) rather than
-- widen the record_index_fields grant.
--
--   published_by_system  NULL, or 'onboarding' (closed vocabulary, the same
--                        word audit_log.actor_system uses for onboarding).
--   published_by_check   exactly one of published_by / published_by_system
--                        is set, or the version is 1. v1 stays as today: any
--                        combination, so every seeded v1 row (published_by
--                        NULL) and every API-published v1 still passes.
--
-- Expand step, safe online:
--   - ADD COLUMN with no DEFAULT is a catalogue change: no rewrite, every
--     existing row reads NULL.
--   - Both CHECKs are added NOT VALID: no scan under the ACCESS EXCLUSIVE
--     lock, every new row is checked from here on. 20261127110900 validates
--     them under SHARE UPDATE EXCLUSIVE, the split 20261101130000/0100 used.
--   - The old published_by_check is dropped and its replacement added under
--     the same name in this one transaction, so no insert runs unchecked
--     between them. Every existing row passes the new rule: v1 rows pass by
--     version, and every later row has published_by set (the old CHECK,
--     validated) and published_by_system NULL (new column).
--   - lock_timeout makes the migration fail and be re-run instead of
--     queueing reads behind a long publish transaction.
--
-- No grant change: nava_onboard and nava_app hold table-level INSERT on the
-- table (db/grants.yaml), which covers the new column. No index: the column
-- is never filtered on. RLS is unchanged: the table's policies are per row
-- and cover the new column; nava_onboard's insert policy still requires a
-- provisioning tenant.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions ADD COLUMN published_by_system text;
ALTER TABLE entity_definition_versions ADD CONSTRAINT entity_definition_versions_published_by_system_check
    CHECK (published_by_system IN ('onboarding')) NOT VALID;
ALTER TABLE entity_definition_versions DROP CONSTRAINT entity_definition_versions_published_by_check;
ALTER TABLE entity_definition_versions ADD CONSTRAINT entity_definition_versions_published_by_check
    CHECK (version = 1 OR num_nonnulls(published_by, published_by_system) = 1) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

-- Fails (23514) once a version after v1 has a system publisher, as it should:
-- the old rule cannot hold that row.
ALTER TABLE entity_definition_versions DROP CONSTRAINT entity_definition_versions_published_by_check;
ALTER TABLE entity_definition_versions DROP COLUMN published_by_system;
ALTER TABLE entity_definition_versions ADD CONSTRAINT entity_definition_versions_published_by_check
    CHECK (published_by IS NOT NULL OR version = 1);
