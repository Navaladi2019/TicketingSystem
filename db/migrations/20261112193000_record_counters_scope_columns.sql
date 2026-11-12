-- S17 P6-01 (TICK-665), ADR-005 §4.1 and §4.2 step M1: record_counters gains
-- scope_key and period_end. A counter row is then one counter per
-- (tenant, type, scope); the default scope '' is today's one counter per type.
-- Runbook: docs/data/record-counters-scope-key.md.
--
-- Expand step, safe online, and R0 (today's binary) keeps working:
--   - Both ADD COLUMNs are catalogue changes: scope_key's constant DEFAULT ''
--     is kept in pg_attribute (PostgreSQL 11+), period_end has no default.
--     No rewrite, no backfill: every existing row reads scope_key = ''.
--     R0's INSERT names neither column, so it writes '' and NULL.
--   - The CHECK is added NOT VALID, so there is no scan under the ACCESS
--     EXCLUSIVE lock; 20261112193100 validates it under SHARE UPDATE
--     EXCLUSIVE (the split of 20261110040000/0100, D10).
--   - lock_timeout: a queued ACCESS EXCLUSIVE would stall every create behind
--     it, so the migration fails and is re-run instead.
--
-- The primary key stays (tenant_id, entity_definition_id) until M3 (S18).
-- Until then a non-empty scope_key for a type that already has a row fails on
-- record_counters_pkey, which is what keeps R0 and R1 on one counter.
--
-- Grants unchanged: nava_app and nava_worker hold table-level INSERT, so the
-- upsert's INSERT arm can set both columns, and UPDATE (next) only, so its
-- UPDATE arm can touch neither (db/grants.yaml). RLS unchanged: the policies
-- are per row and cover the new columns. No audit change: the table is
-- exempt (db/audited-tables.yaml).

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE record_counters
    ADD COLUMN scope_key text NOT NULL DEFAULT '',
    ADD COLUMN period_end timestamptz NULL;
ALTER TABLE record_counters ADD CONSTRAINT record_counters_scope_key_len
    CHECK (length(scope_key) <= 128) NOT VALID;

-- +goose Down

-- Only before M3 (S18): after it the primary key needs scope_key. Run it only
-- once no R1 binary serves (R1's three-column ON CONFLICT needs the column),
-- and after 20261112193200's Down. Dropping the column drops its CHECK.
SET LOCAL lock_timeout = '5s';

ALTER TABLE record_counters
    DROP COLUMN period_end,
    DROP COLUMN scope_key;
