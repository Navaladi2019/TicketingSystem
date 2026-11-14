-- S17 P6-01 (TICK-665), ADR-005 §4.2 step M2: the unique index R1's
-- three-column upsert names as its arbiter,
--   ON CONFLICT (tenant_id, entity_definition_id, scope_key).
-- M3 (S18) promotes it with ADD PRIMARY KEY USING INDEX, so the name
-- record_counters_scoped_key is part of the contract: do not rename it.
--
-- Built CONCURRENTLY: the table takes a write per create, and a plain CREATE
-- INDEX would block them for the build. The index is strictly finer than
-- record_counters_pkey, so the existing rows always fit it. Between M2 and
-- M3 both are enforced: R0's ON CONFLICT (tenant_id, entity_definition_id)
-- arbitrates on the primary key, R1's on this index, and both advance the
-- same row (TestP601CounterScopeExpandWindow).
--
-- Re-runnable (TestNoTransactionMigrationsAreIdempotent). A failed CONCURRENTLY
-- build leaves an INVALID index under the name, which IF NOT EXISTS would keep,
-- so the DO block drops an INVALID one first. A VALID one is kept, unlike
-- 00005's unconditional drop: R1's upsert has no other arbiter, so dropping a
-- valid index under a serving R1 would fail every create until the rebuild
-- finished (the pattern of 20261108120100). After M3 renames the index to
-- record_counters_pkey a re-run would build a second copy; M3's own migration
-- is the place to settle that, not this one.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('record_counters_scoped_key') AND NOT indisvalid) THEN
        DROP INDEX record_counters_scoped_key;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS record_counters_scoped_key
    ON record_counters (tenant_id, entity_definition_id, scope_key);

-- +goose Down

-- Only before M3, and only once no R1 binary serves: R1's upsert has no
-- arbiter without it.
DROP INDEX CONCURRENTLY IF EXISTS record_counters_scoped_key;
