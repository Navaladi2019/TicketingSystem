-- P6-06 (TICK-671): validates the previous migration's
-- entity_definition_versions_number_format_check in its own transaction.
-- A CHECK is validated by a heap scan under SHARE UPDATE EXCLUSIVE, so
-- reads and publishes go on while it runs; RLS does not apply to it.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT entity_definition_versions_number_format_check;

-- +goose Down

SELECT 1;
