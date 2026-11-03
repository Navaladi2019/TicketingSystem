-- P5-13b (TICK-452): attach ADR-004 decision 2's commit check to records
-- (PLAN-005; spike S1 item 1 passed, so records keep the trigger and the
-- P3-13 architecture test is not the only guard).
--
-- From here every transaction that inserts, deletes or updates an audited
-- column of a records row must hold, at COMMIT, an audit_log entry with the
-- row's tenant, this transaction's txid and now(), subject_type 'record' and
-- subject_id = the row's id. The record pipeline writes it through
-- audit.Append in the same transaction; a raw write as nava_app or
-- nava_worker fails at COMMIT with audit_missing (CI-A2, records).
--
-- The UPDATE OF list is db/audited-tables.yaml's audited_columns for
-- records, which is every column db/grants.yaml lets a writer UPDATE; CI-A3
-- (internal/db/audited.Check) fails on any other shape, and on the manifest
-- row not being attached: true. records has no DELETE grant; DELETE is in
-- the event list because the shape is the same for every audited table.
--
-- plan_cache_mode: the function's own SET (20261103150000, ADR-004-E3) is
-- the only control. This migration sets and resets nothing.
--
-- In the same change db/grants.yaml revokes INSERT on audit_log_v0 from
-- nava_app and nava_worker, so S9's table is read-only until P5-17 drops it
-- (PLAN-005 D-5-1). The grant generator applies that; it is not a statement
-- here.

-- +goose Up
CREATE CONSTRAINT TRIGGER records_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF version, schema_version, state, requester_id, assignee_id,
        assignment_group_id, updated_at, data, causation_origin_id, causation_depth ON records
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id');

-- +goose Down
DROP TRIGGER records_audit_commit_check ON records;
