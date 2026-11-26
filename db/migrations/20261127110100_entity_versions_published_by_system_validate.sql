-- P7-12a (TICK-1172): validates 20261127110000's two CHECKs in their own
-- transaction. VALIDATE CONSTRAINT scans the table under SHARE UPDATE
-- EXCLUSIVE, so reads and publishes go on while it runs. No existing row
-- fails: published_by_system is NULL on every row, v1 rows pass by version,
-- and every later row names its principal (the validated P3-06c CHECK).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT entity_definition_versions_published_by_system_check;
ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT entity_definition_versions_published_by_check;

-- +goose Down

-- Nothing to undo: a validated constraint stays valid, and 20261127110000's
-- Down drops both.
