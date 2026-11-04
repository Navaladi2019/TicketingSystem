-- ADR-004-E5 step (ii-b) (TICK-538; D10, CI-E5-8): validates 20261104043000's
-- records_assignment_group_fkey in its own transaction. VALIDATE scans
-- records under SHARE UPDATE EXCLUSIVE and takes ROW SHARE on groups, so
-- record writes and group sync go on while it runs: 4.9 s at 1,000,000
-- records (900,000 with a group) on PostgreSQL 16.10
-- (scratch/e5-groups-contract/). Every assignment group already exists (the
-- old FK held it), so it cannot fail.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;
