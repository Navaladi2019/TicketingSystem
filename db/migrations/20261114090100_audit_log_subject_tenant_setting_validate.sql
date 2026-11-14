-- TICK-854: validates the previous migration's audit_log subject_type CHECK
-- in its own transaction. VALIDATE CONSTRAINT scans under SHARE UPDATE
-- EXCLUSIVE, so reads and audit writes go on while it runs.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log VALIDATE CONSTRAINT audit_log_subject_type_check;

-- +goose Down

SELECT 1;
