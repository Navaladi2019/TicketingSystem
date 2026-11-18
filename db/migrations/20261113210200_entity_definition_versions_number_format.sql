-- P6-06 (TICK-671, ADR-005 §3.2, §3.7 C8, C9): the pinned format link on the
-- immutable entity_definition_versions, never on the mutable
-- entity_definitions row (whose UPDATE grant stays current_version,
-- updated_at, numbering_mode).
--
--   number_format_id, number_format_version  the pinned format version.
--   number_compiled                          the snapshot a create reads (the
--                                            JSON of numbering.Compiled), so
--                                            a create never reads the format
--                                            tables (§3.1).
-- All three NULL means the default numbering, epoch 0. They are set together
-- or not at all (entity_definition_versions_number_format_check).
--
-- The link is a composite foreign key that leads with tenant_id (C8): a
-- tenant A version naming tenant B's format id fails exactly as a made-up id
-- does (23503 on entity_definition_versions_number_format_fkey), so the key
-- is neither a cross-tenant pointer nor an existence oracle. MATCH SIMPLE:
-- tenant_id is never NULL, so MATCH FULL would refuse the default (NULL)
-- link; the CHECK supplies the all-or-none rule instead.
--
-- Locks: ADD COLUMN with no default is catalogue only. The foreign key is
-- added in the same ALTER TABLE that adds its columns, so every existing row
-- has a NULL key column and MATCH SIMPLE checks none of them; it is exempt
-- from the FORCE RLS validation rule (db/fk_validate_force_rls_test.go). The
-- CHECK is added NOT VALID here and validated in the next migration under
-- SHARE UPDATE EXCLUSIVE. lock_timeout makes the migration fail and be re-run
-- instead of queueing publishes behind a long transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions
    ADD COLUMN number_format_id uuid,
    ADD COLUMN number_format_version integer,
    ADD COLUMN number_compiled jsonb,
    ADD CONSTRAINT entity_definition_versions_number_format_fkey
        FOREIGN KEY (tenant_id, number_format_id, number_format_version)
        REFERENCES number_format_versions (tenant_id, format_id, version);
ALTER TABLE entity_definition_versions ADD CONSTRAINT entity_definition_versions_number_format_check CHECK (
    (number_format_id IS NULL) = (number_format_version IS NULL)
    AND (number_format_id IS NULL) = (number_compiled IS NULL)
    AND (number_compiled IS NULL OR jsonb_typeof(number_compiled) = 'object')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions
    DROP CONSTRAINT entity_definition_versions_number_format_check,
    DROP CONSTRAINT entity_definition_versions_number_format_fkey,
    DROP COLUMN number_compiled,
    DROP COLUMN number_format_version,
    DROP COLUMN number_format_id;
