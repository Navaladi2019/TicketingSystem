-- P3-36f (TICK-659): each entity type chooses how its records are numbered
-- (PLAN-003 D-3-9, amended in PR #551): 'gaps' (the default) takes the number
-- in its own short transaction before the create, 'gapless' takes it last
-- inside the create transaction. The allocator (P3-36e, TICK-657) reads the
-- mode from the definition the create path already loads, so this adds no
-- round trip per create.
--
-- Expand step, safe online:
--   - ADD COLUMN with a constant DEFAULT and NOT NULL is a catalogue change
--     since PostgreSQL 11 (the default is kept in pg_attribute and existing
--     rows read it): no table rewrite, no backfill. Every existing type reads
--     'gaps'.
--   - The CHECK is added NOT VALID, so there is no scan under the ACCESS
--     EXCLUSIVE lock, and every new or updated row is checked from here on.
--     20261110040100 validates it under SHARE UPDATE EXCLUSIVE. This is the
--     same split as entity_definition_versions_published_by_check
--     (20261101130000/0100).
--   - The audit commit check is replaced with one whose UPDATE OF list adds
--     numbering_mode. db/grants.yaml gives nava_app UPDATE on the column
--     (P3-36g changes the mode through the type admin API), and CI-A3 requires
--     every updatable column to be audited (db/audited-tables.yaml), so a mode
--     change cannot commit without its config entry. The drop and the create
--     run in this transaction, so no write runs between them unchecked.
--   - Every statement takes its lock on entity_definitions only. The lock is
--     held for the catalogue changes, a few milliseconds. lock_timeout makes
--     the migration fail and be re-run instead of queueing reads behind a
--     long publish transaction.
--
-- No index: the column is read only with the row the create path already
-- fetches by (tenant_id, key). RLS is unchanged: the table's tenant policies
-- are per row and cover the new column.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definitions ADD COLUMN numbering_mode text NOT NULL DEFAULT 'gaps';
ALTER TABLE entity_definitions ADD CONSTRAINT entity_definitions_numbering_mode_check
    CHECK (numbering_mode IN ('gaps', 'gapless')) NOT VALID;

DROP TRIGGER entity_definitions_audit_commit_check ON entity_definitions;
CREATE CONSTRAINT TRIGGER entity_definitions_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF current_version, updated_at, numbering_mode ON entity_definitions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('entity_definition', 'id');

-- +goose Down

SET LOCAL lock_timeout = '5s';

-- The trigger depends on the column, so it goes back first. Dropping the
-- column drops its CHECK and nava_app's column grant with it.
DROP TRIGGER entity_definitions_audit_commit_check ON entity_definitions;
CREATE CONSTRAINT TRIGGER entity_definitions_audit_commit_check
    AFTER INSERT OR DELETE OR UPDATE OF current_version, updated_at ON entity_definitions
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION audit_commit_check('entity_definition', 'id');
ALTER TABLE entity_definitions DROP COLUMN numbering_mode;
