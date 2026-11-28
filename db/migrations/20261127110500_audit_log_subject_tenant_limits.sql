-- TICK-751 (S22 P7-11, folded into TICK-916; ADR-006 D12, I8): registers the
-- audit subject type tenant_limits (internal/audit/subjects). An operator's
-- change to a tenant's limits row is audited under it: domain config,
-- subject_id the tenant id, version_before / version_after NULL (the row is
-- not versioned), content_sha256 the SHA-256 of the after document, e.g.
-- {"filterable_fields_per_type": 10, "reason": "..."} (R3).
--
-- audit_log_subject_type_check mirrors the registry exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'tenant_limits'. The new check is weaker than the old one, so every
-- row satisfies it; it is added NOT VALID here (catalogue only) and validated
-- by the next migration under SHARE UPDATE EXCLUSIVE, which blocks neither
-- reads nor audit writes.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'workflow_definition', 'number_format',
    'tenant_setting', 'tenant_limits', 'audit_export', 'audit_chain', 'erasure_request')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'workflow_definition', 'number_format',
    'tenant_setting', 'audit_export', 'audit_chain', 'erasure_request')) NOT VALID;
