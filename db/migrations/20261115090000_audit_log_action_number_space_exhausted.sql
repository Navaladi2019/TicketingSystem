-- TICK-871 (ADR-004-E9, PLAN-005-E9 T-E9-1): registers the audit action
-- number_space_exhausted. A create refused because its number format's scope
-- has no number left is audited under it (T-E9-2 writes the row).
--
-- audit_log_action_check mirrors audit.Actions() exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'number_space_exhausted'. The new check is weaker than the old one,
-- so every existing row satisfies it, but ADD CONSTRAINT would still scan
-- every partition under ACCESS EXCLUSIVE. It is added NOT VALID here
-- (catalogue only; ALTER TABLE on the partitioned parent reaches every
-- partition) and validated by the next migration under SHARE UPDATE
-- EXCLUSIVE, which blocks neither reads nor audit writes.
--
-- Down refuses while any number_space_exhausted row exists [PG 2]: the old
-- CHECK would not hold for it, and audit rows are never rewritten. The
-- migration role is NOBYPASSRLS and audit_log is FORCE ROW LEVEL SECURITY
-- with no policy for it, so a plain SELECT would see no rows and the guard
-- would always pass. Down lifts FORCE for the check and restores it in the
-- same transaction; the ACCESS EXCLUSIVE lock this takes is the one DROP
-- CONSTRAINT takes anyway, held here through one scan of audit_log (Down is
-- an operator rollback, not an online step).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN (
    'create', 'update', 'publish', 'enable', 'disable', 'delete', 'promote', 'rollback',
    'state_change', 'export', 'erase', 'reseed', 'cascade_cycle', 'cascade_too_deep',
    'number_space_exhausted')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log NO FORCE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM audit_log WHERE action = 'number_space_exhausted') THEN
        RAISE EXCEPTION 'audit_log has number_space_exhausted rows: the 14-action CHECK cannot be restored, and audit rows are never rewritten'
            USING ERRCODE = 'object_in_use';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;

ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN (
    'create', 'update', 'publish', 'enable', 'disable', 'delete', 'promote', 'rollback',
    'state_change', 'export', 'erase', 'reseed', 'cascade_cycle', 'cascade_too_deep')) NOT VALID;
