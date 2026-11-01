-- TICK-378: validates the previous migration's fixed_columns and states
-- CHECKs in their own transaction. VALIDATE CONSTRAINT scans the table under
-- SHARE UPDATE EXCLUSIVE, so reads and publishes go on while it runs.
--
-- The API has always written objects, so a deployed database has nothing to
-- reject. A long-lived database built from the story branch and fed by an
-- old isolation seed ('[]') fails here, naming the constraint; rebuild it
-- (story-branch databases are rebuilt, db/migrations/README.md).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT entity_definition_versions_fixed_columns_check;
ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT entity_definition_versions_states_check;
