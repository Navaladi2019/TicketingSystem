-- P3-35 (TICK-620): attach the audit commit check to acl_versions (ADR-004
-- CI-A3). From here a raw INSERT into acl_versions fails at COMMIT with
-- audit_missing, unless the same transaction wrote an audit_log entry for the
-- same entity type (subject_type entity_definition, subject_id
-- entity_definition_id, db/audited-tables.yaml).
--
-- The two writers of acl_versions are the publish route (authz.PolicyStore.
-- Publish, which writes the entry with the version) and the default incident
-- ACL that onboarding and `nava-onboard reconcile-default-acl` publish through
-- the same function. acl_versions is insert-only, so the trigger has no UPDATE
-- event. The table has no rows in any deployed database (S10 has not
-- shipped), and the trigger only checks rows written after it exists.
--
-- Lock: CREATE CONSTRAINT TRIGGER takes SHARE ROW EXCLUSIVE on acl_versions,
-- which blocks writers of this one table for the few milliseconds the catalogue
-- change takes; reads continue. lock_timeout bounds the wait.
--
-- This migration changes the manifest's `attached` flag for acl_versions to
-- true; that is a security-review change (D-P10).

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE CONSTRAINT TRIGGER acl_versions_audit_commit_check
    AFTER INSERT OR DELETE ON acl_versions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('entity_definition', 'entity_definition_id');

-- +goose Down

DROP TRIGGER acl_versions_audit_commit_check ON acl_versions;
