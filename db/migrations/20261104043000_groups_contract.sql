-- ADR-004-E5 step (ii-a), contract (TICK-538; D2, D10). Runs only after step
-- (i)'s sync (TICK-535), which tombstones a group instead of deleting it, is
-- the only sync running: with DELETE revoked, an old sync that still deletes
-- would fail every tenant's transaction (PLAN-005-E5, the one risk).
--
-- records_assignment_group_fkey becomes ON DELETE RESTRICT. It was ON DELETE
-- SET NULL (assignment_group_id), an FK action that rewrites an audited table
-- with no audit entry (R29). The DROP and ADD are one ALTER TABLE, so no
-- moment exists with no FK. The new FK is NOT VALID: adding it scans nothing,
-- and 20261104043100 validates it under SHARE UPDATE EXCLUSIVE (I5, CI-E5-8).
--
-- Locks, measured on PostgreSQL 16.10 with 50,000 groups and 1,000,000
-- records (scratch/e5-groups-contract/): DROP CONSTRAINT takes ACCESS
-- EXCLUSIVE on records and on groups, held to COMMIT; the ALTER TABLE runs in
-- 13.9 ms. lock_timeout makes the migration fail (55P03) and be re-run rather
-- than queue every read and write behind a long transaction.
--
-- nava_groupsync loses DELETE on groups and its groups_delete policy goes
-- with it (db/grants.yaml). No role but the owner can delete a group now; the
-- owner's delete of a group a record names fails with 23503 (CI-E5-2). The
-- group_members ON DELETE CASCADE FK stays: group_members is not audited.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE records
    DROP CONSTRAINT records_assignment_group_fkey,
    ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY (tenant_id, assignment_group_id)
        REFERENCES groups (tenant_id, id) ON DELETE RESTRICT NOT VALID;

REVOKE DELETE ON groups FROM nava_groupsync;

DROP POLICY groups_delete ON groups;
