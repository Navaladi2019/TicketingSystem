-- P3-40 (TICK-709): registers the audit subject type workflow_definition
-- (internal/audit/subjects), the subject of workflow_definitions and
-- workflow_definition_versions (db/audited-tables.yaml), as ADR-004's
-- Consequences ask of each configuration epic.
--
-- audit_log_subject_type_check mirrors the registry exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'workflow_definition'. The new check is weaker than the old one, so
-- every existing row satisfies it; it is added NOT VALID here (catalogue
-- only) and validated by the next migration under SHARE UPDATE EXCLUSIVE,
-- which blocks neither reads nor audit writes (as 20261109130000/0100).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'workflow_definition', 'audit_export', 'audit_chain',
    'erasure_request')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'audit_export', 'audit_chain', 'erasure_request')) NOT VALID;
