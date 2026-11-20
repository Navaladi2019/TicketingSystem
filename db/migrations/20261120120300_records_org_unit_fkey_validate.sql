-- TICK-947 (S31 P11-14): validates 20261120120200's records_org_unit_fkey in
-- its own transaction, with the TICK-715 pattern
-- (docs/data/fk-validate-force-rls.md): under FORCE ROW LEVEL SECURITY the
-- owner's foreign-key check runs under both tables' RLS, and nava_migrator
-- has no policy on records or org_units, so a bare VALIDATE would see no row.
-- nava_migrator gets a SELECT USING (true) policy on both tables for this
-- transaction only; both are dropped before COMMIT, and a failure rolls them
-- back with everything else (db/fk_validate_force_rls_test.go).
--
-- Every row that existed before 20261120120200 has org_unit_id NULL, which
-- MATCH SIMPLE never checks, and every later row was checked on write, so it
-- cannot fail.
--
-- Locks: CREATE POLICY takes ACCESS EXCLUSIVE on records and org_units, held
-- to COMMIT, so record reads and writes wait for the scan of records (the
-- anti-join skips rows whose org_unit_id is NULL but still reads them): see
-- docs/data/org-units.md for the measured time. lock_timeout makes the
-- migration fail and be re-run instead of queueing behind a long transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE POLICY records_migrator_validate ON records FOR SELECT TO nava_migrator USING (true);
CREATE POLICY org_units_migrator_validate ON org_units FOR SELECT TO nava_migrator USING (true);
ALTER TABLE records VALIDATE CONSTRAINT records_org_unit_fkey;
DROP POLICY org_units_migrator_validate ON org_units;
DROP POLICY records_migrator_validate ON records;

-- +goose Down

SELECT 1;
