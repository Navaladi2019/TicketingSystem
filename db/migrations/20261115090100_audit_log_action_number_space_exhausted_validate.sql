-- TICK-871: validates the previous migration's audit_log action CHECK in its
-- own transaction. VALIDATE CONSTRAINT scans under SHARE UPDATE EXCLUSIVE, so
-- reads and audit writes go on while it runs (about 0.2 s per million rows,
-- ADR-004-E9).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log VALIDATE CONSTRAINT audit_log_action_check;

-- +goose Down

SELECT 1;
