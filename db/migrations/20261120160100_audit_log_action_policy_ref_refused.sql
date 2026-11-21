-- TICK-1025 (S32, ADR-010 D9): registers the audit action
-- policy_ref_refused. A workflow condition step whose named-policy
-- reference is refused (the policy is missing, voided, or of the wrong
-- type) is audited under it (TICK-957 writes the row), so it no longer reads
-- as an access-policy update (config / update / acl_policy).
--
-- audit_log_action_check mirrors audit.Actions() exactly
-- (TestP510bSubjectTypesMirrorRegistry), so it is replaced by the same list
-- plus 'policy_ref_refused'. The new check is weaker than the old one, so
-- every existing row satisfies it, but ADD CONSTRAINT would still scan every
-- partition under ACCESS EXCLUSIVE. It is added NOT VALID here (catalogue
-- only; ALTER TABLE on the partitioned parent reaches every partition) and
-- validated by the next migration under SHARE UPDATE EXCLUSIVE, which blocks
-- neither reads nor audit writes. Re-running Up converges to the same CHECK.
--
-- Down refuses while any policy_ref_refused row exists: the old CHECK would
-- not hold for it, and audit rows are never rewritten. As in
-- 20261115090000, Down lifts FORCE ROW LEVEL SECURITY for the guard (the
-- migration role has no audit_log policy, so a plain SELECT would see no
-- rows) and restores it in the same transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN (
    'create', 'update', 'publish', 'enable', 'disable', 'delete', 'promote', 'rollback',
    'state_change', 'export', 'erase', 'reseed', 'cascade_cycle', 'cascade_too_deep',
    'number_space_exhausted', 'policy_ref_refused')) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE audit_log NO FORCE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM audit_log WHERE action = 'policy_ref_refused') THEN
        RAISE EXCEPTION 'audit_log has policy_ref_refused rows: the 15-action CHECK cannot be restored, and audit rows are never rewritten'
            USING ERRCODE = 'object_in_use';
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;

ALTER TABLE audit_log DROP CONSTRAINT audit_log_action_check;
ALTER TABLE audit_log ADD CONSTRAINT audit_log_action_check CHECK (action IN (
    'create', 'update', 'publish', 'enable', 'disable', 'delete', 'promote', 'rollback',
    'state_change', 'export', 'erase', 'reseed', 'cascade_cycle', 'cascade_too_deep',
    'number_space_exhausted')) NOT VALID;
