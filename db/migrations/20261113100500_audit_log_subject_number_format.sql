-- P6-06 (TICK-671, ADR-005 §3.7 C12): registers the audit subject type
-- number_format (internal/audit/subjects). A format publish is audited under
-- its own subject, keyed by the format id.
--
-- audit_log_subject_type_check mirrors the registry exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'number_format'. The new check is weaker than the old one, so every
-- existing row satisfies it, but ADD CONSTRAINT would still scan every
-- partition under ACCESS EXCLUSIVE. It is added NOT VALID here (catalogue
-- only) and validated by the next migration under SHARE UPDATE EXCLUSIVE,
-- which blocks neither reads nor audit writes.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'number_format', 'audit_export', 'audit_chain',
    'erasure_request')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'audit_export', 'audit_chain', 'erasure_request')) NOT VALID;
