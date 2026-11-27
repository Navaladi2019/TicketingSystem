-- TICK-1158 (ADR-010-A4 A4-1): records_list_updated also stores org_unit_id
-- as an INCLUDE column, so shape R's unfiltered list count is an Index Only
-- Scan of the type's index range instead of a read of the heap. The key, and
-- so the list sort it serves (TICK-640), is unchanged. org_unit_id is a fixed
-- column: no expression index and nothing on data (PLAN-003 D25 cut).
--
-- Online: the new definition is built CONCURRENTLY as records_list_updated_v2,
-- the old index is dropped CONCURRENTLY, then v2 takes its name. Each step is
-- safe to run again (TestNoTransactionMigrationsAreIdempotent):
--   - the DO block drops a v2 that is INVALID (a failed build) or does not
--     have the target definition (left by a failed Down), as
--     20261109140000_records_list_updated.sql does for its index;
--   - if records_list_updated already has the target definition (the swap
--     finished), the DO block renames it to v2, so the build below is a no-op
--     and the rename puts the name back: a re-run builds nothing;
--   - a run that stopped after the drop finishes with the rename alone.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    -- v2 is kept only when it is valid and covering (INCLUDE: more attributes than key attributes).
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_list_updated_v2')
               AND NOT (indisvalid AND indnatts > indnkeyatts)) THEN
        DROP INDEX records_list_updated_v2;
    END IF;
    IF to_regclass('records_list_updated_v2') IS NULL
       AND EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_list_updated')
                   AND indisvalid AND indnatts > indnkeyatts) THEN
        ALTER INDEX records_list_updated RENAME TO records_list_updated_v2;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_list_updated_v2
    ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC) INCLUDE (org_unit_id);
DROP INDEX CONCURRENTLY IF EXISTS records_list_updated;
ALTER INDEX IF EXISTS records_list_updated_v2 RENAME TO records_list_updated;

-- +goose Down

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_list_updated_v2')
               AND NOT (indisvalid AND indnatts = indnkeyatts)) THEN
        DROP INDEX records_list_updated_v2;
    END IF;
    IF to_regclass('records_list_updated_v2') IS NULL
       AND EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_list_updated')
                   AND indisvalid AND indnatts = indnkeyatts) THEN
        ALTER INDEX records_list_updated RENAME TO records_list_updated_v2;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_list_updated_v2
    ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);
DROP INDEX CONCURRENTLY IF EXISTS records_list_updated;
ALTER INDEX IF EXISTS records_list_updated_v2 RENAME TO records_list_updated;
