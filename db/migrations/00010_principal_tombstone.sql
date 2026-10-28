-- TICK-220 (T25 follow-up): a principal who leaves the Keycloak directory is
-- tombstoned, not deleted, so a foreign key to principals never makes group
-- sync's DELETE fail and roll the tenant's sync back on every cycle (QA
-- finding 2, docs/reviews/tick-78-groupsync-worker.md). Decision:
-- docs/delivery/decision-log.md, 2026-10-27.
--
-- deleted_at NULL = in the directory; set = removed at that time. The CHECK
-- makes a tombstone disabled, so the resolver's existing enabled test refuses
-- it (401) even if a reader forgets deleted_at. A user who comes back keeps
-- the same row and id (UNIQUE (tenant_id, keycloak_user_id)): sync clears
-- deleted_at, and anything that pointed at them points at them again.
--
-- Expand step only. Group sync still deletes until its own task switches it
-- to an UPDATE; the contract step (revoke DELETE from nava_groupsync, drop
-- principals_delete) follows that change. No index: nothing reads by
-- deleted_at, and the resolver reads by the unique Keycloak id.
--
-- Both statements are catalogue changes under a short ACCESS EXCLUSIVE lock
-- (3 ms on the 519,000-row identity bench). The CHECK is NOT VALID here and
-- validated by 00011 under SHARE UPDATE EXCLUSIVE, which blocks neither reads
-- nor writes: validating here would hold ACCESS EXCLUSIVE for the whole scan,
-- 95-143 ms over four runs on the bench, and growing with the table
-- (docs/data/principal-tombstone.md). New and updated rows are checked from
-- this migration on. lock_timeout keeps the migration from queueing reads
-- behind a long sync transaction: it fails and is re-run instead.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE principals ADD COLUMN deleted_at timestamptz;
ALTER TABLE principals ADD CONSTRAINT principals_tombstone_check
    CHECK (deleted_at IS NULL OR NOT enabled) NOT VALID;
