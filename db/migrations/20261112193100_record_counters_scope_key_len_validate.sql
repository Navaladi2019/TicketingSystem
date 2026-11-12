-- S17 P6-01 (TICK-665), ADR-005 §4.2 step M1, second half: validates
-- record_counters_scope_key_len, added NOT VALID by 20261112193000. VALIDATE
-- CONSTRAINT takes SHARE UPDATE EXCLUSIVE, which blocks neither the upsert's
-- INSERT nor its UPDATE, while it scans the existing rows (all read '').

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE record_counters VALIDATE CONSTRAINT record_counters_scope_key_len;

-- +goose Down

-- Nothing to undo: a validated constraint stays valid, and 20261112193000's
-- Down drops it with the column.
