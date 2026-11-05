-- P5-13c (TICK-530): attach ADR-004 decision 2's commit check to records,
-- after ADR-004-E5 step (ii) (D10). TICK-538 made
-- records_assignment_group_fkey ON DELETE RESTRICT (20261104043000, validated
-- in 20261104043100) and took DELETE on groups away from group sync, which
-- now tombstones a group Keycloak drops (TICK-535). No FK action rewrites a
-- records row any more (R29, checked by CI-A3), so no role that cannot write
-- an audit entry ever writes records. That was the TICK-452 stall
-- (D-5-14) this attach waited for.
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
-- CREATE CONSTRAINT TRIGGER takes SHARE ROW EXCLUSIVE on records for a
-- catalogue change only (no scan, no rewrite); lock_timeout bounds the wait
-- behind a long record transaction.

-- +goose Up
SET LOCAL lock_timeout = '5s';
CREATE CONSTRAINT TRIGGER records_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF version, schema_version, state, requester_id, assignee_id,
        assignment_group_id, updated_at, data, causation_origin_id, causation_depth ON records
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id');

-- +goose Down
DROP TRIGGER records_audit_commit_check ON records;
