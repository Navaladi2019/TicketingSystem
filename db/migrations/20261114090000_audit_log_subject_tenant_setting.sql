-- TICK-854 (for TICK-672, ADR-005 §3.6): registers the audit subject type
-- tenant_setting (internal/audit/subjects). A change to a tenant-level
-- setting (first: tenants.time_zone) is audited under it: domain config,
-- subject_id the tenant id, version_before / version_after NULL (settings
-- are not versioned), content_sha256 the SHA-256 of the after document,
-- e.g. {"time_zone": "Asia/Kolkata"} (R3).
--
-- audit_log_subject_type_check mirrors the registry exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'tenant_setting'. The new check is weaker than the old one, so every
-- existing row satisfies it, but ADD CONSTRAINT would still scan every
-- partition under ACCESS EXCLUSIVE. It is added NOT VALID here (catalogue
-- only) and validated by the next migration under SHARE UPDATE EXCLUSIVE,
-- which blocks neither reads nor audit writes.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'number_format', 'tenant_setting', 'audit_export',
    'audit_chain', 'erasure_request')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'number_format', 'audit_export', 'audit_chain',
    'erasure_request')) NOT VALID;
