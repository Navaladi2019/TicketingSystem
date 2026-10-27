-- TICK-220: validates 00006's principals_tombstone_check in its own
-- transaction. VALIDATE CONSTRAINT scans the table under SHARE UPDATE
-- EXCLUSIVE, so reads and sync writes go on while it runs (86-172 ms on the
-- 519,000-row identity bench, docs/data/principal-tombstone.md). Every
-- existing row has deleted_at NULL, so it cannot fail.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE principals VALIDATE CONSTRAINT principals_tombstone_check;
