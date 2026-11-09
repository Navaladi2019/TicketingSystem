-- P3-35 (TICK-620), contract amendment A3: registers the audit subject type
-- acl_policy (internal/audit/subjects). An ACL publish is audited under its own
-- subject, keyed by the entity definition id, so its versions never share a
-- (subject, version) key with the entity type's schema versions.
--
-- audit_log_subject_type_check mirrors the registry exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'acl_policy'. The new check is weaker than the old one, so every
-- existing row satisfies it, but ADD CONSTRAINT would still scan every
-- partition under ACCESS EXCLUSIVE. It is added NOT VALID here (catalogue
-- only) and validated by the next migration under SHARE UPDATE EXCLUSIVE, which
-- blocks neither reads nor audit writes.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'acl_policy', 'audit_export', 'audit_chain', 'erasure_request')) NOT VALID;

-- +goose Down

ALTER TABLE audit_log DROP CONSTRAINT audit_log_subject_type_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_subject_type_check CHECK (subject_type IN (
    'record', 'cascade_key', 'entity_definition', 'audit_export', 'audit_chain', 'erasure_request')) NOT VALID;
