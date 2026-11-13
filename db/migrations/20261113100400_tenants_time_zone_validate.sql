-- P6-06 (TICK-671): validates tenants_time_zone_check in its own
-- transaction, under SHARE UPDATE EXCLUSIVE.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE tenants VALIDATE CONSTRAINT tenants_time_zone_check;

-- +goose Down

SELECT 1;
