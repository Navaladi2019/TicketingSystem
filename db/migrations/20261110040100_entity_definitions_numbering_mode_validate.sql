-- P3-36f (TICK-659): validates entity_definitions_numbering_mode_check, added
-- NOT VALID by 20261110040000. VALIDATE CONSTRAINT takes SHARE UPDATE
-- EXCLUSIVE, which blocks neither reads nor writes of the table, while it
-- scans the existing rows (one per entity type; all read the 'gaps' default).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE entity_definitions VALIDATE CONSTRAINT entity_definitions_numbering_mode_check;

-- +goose Down

-- Nothing to undo: a validated constraint stays valid, and 20261110040000's
-- Down drops it with the column.

