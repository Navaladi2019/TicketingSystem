-- TICK-265 (contract step of TICK-220): group sync tombstones a principal
-- that leaves the directory instead of deleting it, so nava_groupsync no
-- longer needs DELETE on principals. db/grants.yaml revokes the privilege;
-- this drops the policy that allowed it. Decision: docs/delivery/decision-log.md,
-- 2026-10-27; plan: docs/data/principal-tombstone.md, step 3.
--
-- A catalogue change under a short lock, no row is touched. IF EXISTS makes a
-- re-run safe.

-- +goose Up

SET LOCAL lock_timeout = '5s';

DROP POLICY IF EXISTS principals_delete ON principals;
