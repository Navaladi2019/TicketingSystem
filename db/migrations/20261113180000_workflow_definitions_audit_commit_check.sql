-- P3-41b (TICK-737): attach the audit commit check to workflow_definitions and
-- workflow_definition_versions (ADR-004 CI-A3), as P3-40's migration said
-- P3-41 would. From here a raw INSERT or DELETE of a version, or an INSERT,
-- DELETE or UPDATE of a definition's current_version or updated_at, fails at
-- COMMIT with audit_missing unless the same transaction wrote an audit_log
-- entry for the definition (subject_type workflow_definition, subject_id the
-- definition's id; db/audited-tables.yaml).
--
-- The one runtime writer is workflowdefs.Store.Publish (P3-41), which writes
-- its config entry in the publish transaction, one per version. It creates
-- the definition and its version 1 in that transaction, so one entry covers
-- both rows. Fixtures that need a raw row write their own entry.
--
-- Both tables hold only rows written since P3-40 shipped; the trigger checks
-- only rows written after it exists. Lock: CREATE CONSTRAINT TRIGGER takes
-- SHARE ROW EXCLUSIVE on its table for a catalogue change only; lock_timeout
-- bounds the wait behind a long writer.
--
-- This migration changes the manifest's `attached` flag for both tables to
-- true; that is a security-review change (D-P10).

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE CONSTRAINT TRIGGER workflow_definitions_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF current_version, updated_at ON workflow_definitions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('workflow_definition', 'id');

CREATE CONSTRAINT TRIGGER workflow_definition_versions_audit_commit_check
    AFTER INSERT OR DELETE ON workflow_definition_versions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('workflow_definition', 'definition_id');

-- +goose Down

SET LOCAL lock_timeout = '5s';

DROP TRIGGER workflow_definition_versions_audit_commit_check ON workflow_definition_versions;
DROP TRIGGER workflow_definitions_audit_commit_check ON workflow_definitions;
