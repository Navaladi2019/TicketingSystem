-- TICK-715: re-checks records_assignment_group_fkey for real. It fails, and
-- changes nothing, if a record names an assignment group its tenant does not
-- hold.
--
-- 20261104043100 validated this key with a bare VALIDATE, which checked
-- nothing: under FORCE ROW LEVEL SECURITY the owner's foreign-key check is
-- one anti-join query (RI_Initial_Check) that runs under both tables' RLS,
-- and nava_migrator has no policy on records or groups, so it saw no row
-- (docs/data/fk-validate-force-rls.md). A key already marked valid is not
-- re-checked by VALIDATE, so this transaction drops it and adds it back NOT
-- VALID, with the same definition (ON DELETE RESTRICT), before validating.
--
-- The P3-40 pattern: nava_migrator gets a SELECT USING (true) policy on both
-- tables for this transaction only. Both are dropped before COMMIT, and a
-- failure rolls them back with everything else, so no other session ever
-- sees them; db/fk_validate_force_rls_test.go fails a migration that skips
-- the pattern or leaves a policy behind.
--
-- Locks: CREATE POLICY and DROP CONSTRAINT take ACCESS EXCLUSIVE on records
-- and groups, held to COMMIT, so record and group reads and writes wait for
-- the scan: 0.58 s at 50,000 groups and 1,000,000 records (900,000 with a
-- group) on PostgreSQL 16.10 (scratch/tick-715-fk-recheck/). lock_timeout
-- makes the migration fail and be re-run instead of queueing behind a long
-- transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE POLICY records_migrator_validate ON records FOR SELECT TO nava_migrator USING (true);
CREATE POLICY groups_migrator_validate ON groups FOR SELECT TO nava_migrator USING (true);
ALTER TABLE records
    DROP CONSTRAINT records_assignment_group_fkey,
    ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
        REFERENCES groups (tenant_id, id) ON DELETE RESTRICT NOT VALID;
ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
DROP POLICY groups_migrator_validate ON groups;
DROP POLICY records_migrator_validate ON records;

-- +goose Down

SELECT 1;
