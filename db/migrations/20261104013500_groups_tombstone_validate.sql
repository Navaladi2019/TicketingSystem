-- ADR-004-E5 step (i) (TICK-534, I5): validates 20261104013400's
-- group_members_live_check and group_members_group_live_fkey in their own
-- transaction. VALIDATE scans group_members under SHARE UPDATE EXCLUSIVE and
-- takes ROW SHARE on groups, so reads and sync writes go on while it runs:
-- 171 ms for the CHECK and 742 ms for the FK at 2,500,000 memberships on
-- PostgreSQL 16.10 (scratch/e5-groups-tombstone/). Every existing membership
-- has live = true and every group deleted_at NULL, so it cannot fail.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE group_members VALIDATE CONSTRAINT group_members_live_check;
ALTER TABLE group_members VALIDATE CONSTRAINT group_members_group_live_fkey;
