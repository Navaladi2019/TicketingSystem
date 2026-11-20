-- TICK-947 (S31 P11-14; ADR-010 D7, D8; PLAN-011 C5): records.org_unit_id,
-- the unit a record belongs to for the under leaf. The write pipeline sets it
-- on create from the requester's primary unit (TICK-949); after that it is a
-- stored field, moved only by a write (a workflow step), never recomputed on
-- read. NULL = no unit, and NULL never matches under.
--
-- Expand step, safe online:
--   - ADD COLUMN with no default is a catalogue change: no rewrite, every
--     existing row reads NULL.
--   - records_org_unit_fkey (tenant_id, org_unit_id) -> org_units (tenant_id,
--     id) carries the tenant: a record can name only its own tenant's unit.
--     NO ACTION, no ON DELETE: units are tombstoned, never deleted (D7), and
--     a tombstoned unit stays a valid target. Added NOT VALID (no scan under
--     ACCESS EXCLUSIVE); new and updated rows are checked from here on, and
--     20261120120300 validates it with the TICK-715 pattern.
--   - The audit commit check is replaced with one whose UPDATE OF list adds
--     org_unit_id: db/grants.yaml gives nava_app and nava_worker UPDATE on the
--     column (D8: moved by a workflow step), and CI-A3 requires every
--     updatable column to be audited (db/audited-tables.yaml). The drop and
--     the create run in this transaction, so no write runs between them
--     unchecked.
--   - Every statement locks records (and org_units for the FK, SHARE ROW
--     EXCLUSIVE) for catalogue changes only, a few milliseconds. lock_timeout
--     makes the migration fail and be re-run instead of queueing behind a
--     long record transaction.
--
-- The index is 20261120120400 (CREATE INDEX CONCURRENTLY, its own file).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE records
    ADD COLUMN org_unit_id uuid,
    ADD CONSTRAINT records_org_unit_fkey FOREIGN KEY (tenant_id, org_unit_id)
        REFERENCES org_units (tenant_id, id) NOT VALID;

DROP TRIGGER records_audit_commit_check ON records;
CREATE CONSTRAINT TRIGGER records_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF version, schema_version, state, requester_id, assignee_id,
        assignment_group_id, org_unit_id, updated_at, data, causation_origin_id, causation_depth ON records
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id');

-- +goose Down

SET LOCAL lock_timeout = '5s';

-- The trigger depends on the column, so it goes back first. Dropping the
-- column drops its FK and the column grants with it.
DROP TRIGGER records_audit_commit_check ON records;
CREATE CONSTRAINT TRIGGER records_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF version, schema_version, state, requester_id, assignee_id,
        assignment_group_id, updated_at, data, causation_origin_id, causation_depth ON records
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('record', 'id');
ALTER TABLE records DROP COLUMN org_unit_id;
