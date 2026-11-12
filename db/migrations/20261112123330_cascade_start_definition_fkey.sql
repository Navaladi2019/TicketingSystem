-- P3-40 (TICK-709), ADR-001 A5-6: cascade_start.definition_id gets its
-- composite foreign key to workflow_definitions, which the previous
-- migrations created. Composite, because PostgreSQL runs foreign-key checks
-- as the table owner and so ignores RLS: tenant A cannot claim a start of
-- tenant B's definition.
--
-- Expand step, as 00010/00011 (TICK-88 Postgres review): ADD CONSTRAINT ...
-- NOT VALID is a catalogue change under a short SHARE ROW EXCLUSIVE lock on
-- both tables, with no scan; every new claim is checked from here on.
-- 20261112123340 validates the existing rows in its own transaction (and
-- explains why it cannot be a bare VALIDATE: FORCE RLS makes that vacuous).
--
-- Rows claimed before this migration name definitions no table held (the
-- S10/S11a test sinks and the demo commands invented ids). Validation fails on
-- any such row in a long-lived database: delete those claims, or seed the
-- definitions they name, before 20261112123340 runs. No shipped environment
-- has any, since no released build claims a start.
--
-- No index on the referencing side: definitions are never deleted or
-- re-keyed (no DELETE grant, no UPDATE of id), so nothing scans it.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE cascade_start ADD CONSTRAINT cascade_start_definition_fkey FOREIGN KEY (tenant_id, definition_id)
    REFERENCES workflow_definitions (tenant_id, id) NOT VALID;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE cascade_start DROP CONSTRAINT cascade_start_definition_fkey;
