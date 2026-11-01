-- P3-06c (TICK-383): validates 20261101130000's
-- entity_definition_versions_published_by_check in its own transaction.
-- VALIDATE CONSTRAINT scans the table under SHARE UPDATE EXCLUSIVE, so reads
-- and publishes go on while it runs. Onboarding seeds only version 1 with
-- published_by NULL, and Publish always names its publisher, so no existing
-- row fails it.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definition_versions VALIDATE CONSTRAINT entity_definition_versions_published_by_check;
