-- TICK-715: re-checks group_members_group_live_fkey for real. It fails, and
-- changes nothing, if a membership names a group that is missing or
-- tombstoned.
--
-- 20261104013500 validated this key with a bare VALIDATE, which checked
-- nothing: under FORCE ROW LEVEL SECURITY the owner's foreign-key check is
-- one anti-join query (RI_Initial_Check) that runs under both tables' RLS,
-- and nava_migrator has no policy on group_members or groups, so it saw no
-- row (docs/data/fk-validate-force-rls.md). A key already marked valid is not
-- re-checked by VALIDATE, so this transaction drops it and adds it back NOT
-- VALID, with the same definition, before validating.
--
-- The P3-40 pattern: nava_migrator gets a SELECT USING (true) policy on both
-- tables for this transaction only. Both are dropped before COMMIT, and a
-- failure rolls them back with everything else, so no other session ever
-- sees them; db/fk_validate_force_rls_test.go fails a migration that skips
-- the pattern or leaves a policy behind.
--
-- Locks: CREATE POLICY and DROP CONSTRAINT take ACCESS EXCLUSIVE on
-- group_members and groups, held to COMMIT, so membership and group reads
-- wait for the scan: 1.02 s at 50,000 groups and 2,500,000 memberships on
-- PostgreSQL 16.10 (scratch/tick-715-fk-recheck/). lock_timeout makes the
-- migration fail and be re-run instead of queueing behind a long transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE POLICY group_members_migrator_validate ON group_members FOR SELECT TO nava_migrator USING (true);
CREATE POLICY groups_migrator_validate ON groups FOR SELECT TO nava_migrator USING (true);
ALTER TABLE group_members
    DROP CONSTRAINT group_members_group_live_fkey,
    ADD CONSTRAINT group_members_group_live_fkey FOREIGN KEY (tenant_id, group_id, live)
        REFERENCES groups (tenant_id, id, live) NOT VALID;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
DROP POLICY groups_migrator_validate ON groups;
DROP POLICY group_members_migrator_validate ON group_members;

-- +goose Down

SELECT 1;
