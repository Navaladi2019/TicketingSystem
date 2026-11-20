-- TICK-940 (ADR-010-A1 A1-2b): validates the previous migration's two byte
-- ceilings in their own transaction. VALIDATE CONSTRAINT scans the table
-- under SHARE UPDATE EXCLUSIVE, so reads and publishes go on while it runs.
-- masks is checked on every row, format 1 included: a stored row above
-- 131,072 bytes fails here, naming its constraint, instead of blocking
-- later publishes.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE acl_versions VALIDATE CONSTRAINT acl_versions_bytes_rules_check;
ALTER TABLE acl_versions VALIDATE CONSTRAINT acl_versions_bytes_masks_check;

-- +goose Down

-- Nothing: a constraint cannot be made NOT VALID again, and the previous
-- migration's Down drops it.
SELECT 1;
