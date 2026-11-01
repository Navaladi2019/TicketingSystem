-- TICK-378: fixed_columns and states on entity_definition_versions must be
-- JSON objects. The store scans them into map[string]string and
-- metadata.StateModel, and pgx v5 refuses any other top-level type
-- ("cannot unmarshal array"), so one stray row turns every read of its type
-- into a 500. Versions are immutable (D-3-4): no role may UPDATE a bad row
-- afterwards, so the only place to stop it is the insert.
--
-- Top-level type only. The members (string values, initial / states /
-- transitions) are metadata.Compile's job before the insert; repeating
-- those rules in SQL would drift from the Go ones. No index: nothing reads
-- by these predicates.
--
-- Added NOT VALID here, under a short ACCESS EXCLUSIVE lock (catalogue only,
-- no scan), and validated by the next migration under SHARE UPDATE
-- EXCLUSIVE, which blocks neither reads nor publishes. New rows are checked
-- from this migration on. Figures: docs/data/entity-def-versions-jsonb-check.md.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions
    ADD CONSTRAINT entity_definition_versions_fixed_columns_check
        CHECK (jsonb_typeof(fixed_columns) = 'object') NOT VALID,
    ADD CONSTRAINT entity_definition_versions_states_check
        CHECK (jsonb_typeof(states) = 'object') NOT VALID;
