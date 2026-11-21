-- TICK-947 (S31 P11-14; ADR-010 D7; SP-2 docs/data/acl-format2-caps.md
-- section 5): records_org_unit, the key ADR-010 D7 names, (tenant_id,
-- org_unit_id), partial on org_unit_id IS NOT NULL. Tenants without regions
-- leave the column NULL, so the index holds only rows that have a unit. Both
-- the under leaf (org_unit_id IS NOT NULL AND org_unit_id = ANY(S)) and the
-- NO ACTION check of records_org_unit_fkey imply its predicate. Size and plans:
-- docs/data/org-units.md.
--
-- Online: CREATE INDEX CONCURRENTLY, alone in a NO TRANSACTION file. A
-- failed build leaves an INVALID index, which the DO block drops on the
-- re-run (TestNoTransactionMigrationsAreIdempotent); a valid one is kept.

-- +goose NO TRANSACTION
-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_index WHERE indexrelid = to_regclass('records_org_unit') AND NOT indisvalid) THEN
        DROP INDEX records_org_unit;
    END IF;
END
$$;
-- +goose StatementEnd
CREATE INDEX CONCURRENTLY IF NOT EXISTS records_org_unit
    ON records (tenant_id, org_unit_id) WHERE org_unit_id IS NOT NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS records_org_unit;
