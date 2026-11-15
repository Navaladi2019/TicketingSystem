-- P6-06 (TICK-671, ADR-005 §3.6): the tenant's numbering time zone, an IANA
-- name, default UTC. There is no tenant settings table, so it is a column
-- on tenants. A change affects future scope keys only.
--
-- The CHECK pins the shape of an IANA name (Area/Location, Etc/GMT+5, UTC)
-- and its length; whether the zone exists is checked in Go against the
-- tzdata the binary ships with, at the write, because pg_timezone_names is
-- not immutable and so cannot back a CHECK.
--
-- No grant changes: nava_app reads tenants by column (id, state), so the
-- route that reads or sets the time zone adds its own column grant.
--
-- Locks: a constant default is catalogue only, no rewrite. The CHECK is
-- added NOT VALID here and validated in the next migration.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE tenants ADD COLUMN time_zone text NOT NULL DEFAULT 'UTC';
ALTER TABLE tenants ADD CONSTRAINT tenants_time_zone_check
    CHECK (time_zone ~ '^[A-Za-z][A-Za-z0-9_+./-]{0,63}$') NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE tenants DROP COLUMN time_zone;
