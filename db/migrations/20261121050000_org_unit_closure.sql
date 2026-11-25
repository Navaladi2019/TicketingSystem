-- TICK-1032 (S31 fix; ADR-010 D7 as amended by the Solution Architect's R1
-- ruling on TICK-946, ADR-010-A2; PLAN-011): org_unit_closure lists, for
-- each live org unit, every live unit at or below it, so the under leaf reads
-- one table and no request walks the org tree. Platform DDL, no per-tenant
-- DDL. The contract is TICK-1032's; docs/data/org-unit-closure.md has the
-- model and the figures.
--
-- 1. Rows (tenant_id, ancestor_id, descendant_id). Every live unit has its
--    self-row. A row exists only if every unit from the root down to the
--    descendant is live: a tombstoned unit has no rows, and neither has any
--    unit below it, even a live one. A unit on a parent cycle (no root above
--    it) has none either.
--
-- 2. Keys. The primary key (tenant_id, ancestor_id, descendant_id) is the
--    only index: the Allows check probes it, the list leaf reads descendant_id
--    by its (tenant_id, ancestor_id) prefix, and group sync's per-tenant
--    DELETE uses its tenant_id prefix. Both FKs carry tenant_id and reference
--    org_units (tenant_id, id) with no action: units are tombstoned, never
--    deleted (no role holds DELETE on org_units), and id is never updated, so
--    no FK check ever has to find a closure row by descendant_id.
--
-- 3. FORCE RLS with org_units' app.tenant_id policies: SELECT for nava_app,
--    nava_worker and nava_groupsync; INSERT and DELETE for nava_groupsync, the
--    only writer (a rebuild deletes the tenant's rows and inserts them again).
--    No UPDATE grant, so no UPDATE policy.
--
-- 4. Backfill. org_unit_closure_rebuild_all() fills the closure of every
--    tenant from org_units, with the recursive query group sync's rebuild
--    runs (TICK-1033): live roots, then live children of what is reached. It
--    deletes a tenant's rows before inserting them, so a re-run as
--    nava_migrator gives the same rows. Access path (ADR-004 R26): temporary
--    nava_migrator policies created and dropped inside the function, each
--    tenant under set_config('app.tenant_id', ..., true); the only
--    cross-tenant one is SELECT on tenants. The R26.3 check below fails the
--    migration if one is left over.
--
-- Locks: the table is new. org_units is locked in SHARE mode for the
-- backfill, so no sync transaction changes the tree between the read and the
-- commit; reads go on. lock_timeout makes the migration fail and be re-run
-- instead of queueing reads behind a long sync transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

CREATE TABLE org_unit_closure (
    tenant_id     uuid NOT NULL,
    ancestor_id   uuid NOT NULL,
    descendant_id uuid NOT NULL,
    CONSTRAINT org_unit_closure_pkey PRIMARY KEY (tenant_id, ancestor_id, descendant_id),
    CONSTRAINT org_unit_closure_ancestor_fkey FOREIGN KEY (tenant_id, ancestor_id)
        REFERENCES org_units (tenant_id, id),
    CONSTRAINT org_unit_closure_descendant_fkey FOREIGN KEY (tenant_id, descendant_id)
        REFERENCES org_units (tenant_id, id)
);
ALTER TABLE org_unit_closure ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_unit_closure FORCE ROW LEVEL SECURITY;

CREATE POLICY org_unit_closure_select ON org_unit_closure FOR SELECT TO nava_app, nava_worker, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_unit_closure_insert ON org_unit_closure FOR INSERT TO nava_groupsync
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_unit_closure_delete ON org_unit_closure FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementBegin
CREATE FUNCTION org_unit_closure_rebuild_all() RETURNS bigint
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    tid uuid;
    filled bigint := 0;
    n bigint;
BEGIN
    IF current_user <> 'nava_migrator' THEN
        RAISE EXCEPTION 'org_unit_closure_rebuild_all: only nava_migrator may run it, not %', current_user
            USING ERRCODE = '42501';
    END IF;

    CREATE POLICY ouc_backfill_tenants ON tenants FOR SELECT TO nava_migrator
        USING (true);
    CREATE POLICY ouc_backfill_units_select ON org_units FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    -- DELETE ... WHERE reads the rows it deletes, so it needs SELECT too.
    CREATE POLICY ouc_backfill_closure_select ON org_unit_closure FOR SELECT TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY ouc_backfill_closure_insert ON org_unit_closure FOR INSERT TO nava_migrator
        WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
    CREATE POLICY ouc_backfill_closure_delete ON org_unit_closure FOR DELETE TO nava_migrator
        USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

    LOCK TABLE org_units IN SHARE MODE;

    FOR tid IN SELECT id FROM tenants ORDER BY id LOOP
        PERFORM set_config('app.tenant_id', tid::text, true);
        DELETE FROM org_unit_closure WHERE tenant_id = tid;
        -- live: units whose whole path from a root is live. A unit hangs
        -- under one parent, so a walk down from the roots is a tree and ends.
        -- closure: from each live unit, down through live children.
        INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
        WITH RECURSIVE live (id) AS (
            SELECT u.id FROM org_units u
            WHERE u.tenant_id = tid AND u.parent_id IS NULL AND u.deleted_at IS NULL
            UNION ALL
            SELECT u.id FROM live l
            JOIN org_units u ON u.tenant_id = tid AND u.parent_id = l.id
            WHERE u.deleted_at IS NULL
        ), closure (ancestor_id, descendant_id) AS (
            SELECT id, id FROM live
            UNION ALL
            SELECT c.ancestor_id, u.id FROM closure c
            JOIN org_units u ON u.tenant_id = tid AND u.parent_id = c.descendant_id
            WHERE u.deleted_at IS NULL
        )
        SELECT tid, ancestor_id, descendant_id FROM closure;
        GET DIAGNOSTICS n = ROW_COUNT;
        filled := filled + n;
    END LOOP;
    PERFORM set_config('app.tenant_id', '', true);

    DROP POLICY ouc_backfill_closure_delete ON org_unit_closure;
    DROP POLICY ouc_backfill_closure_insert ON org_unit_closure;
    DROP POLICY ouc_backfill_closure_select ON org_unit_closure;
    DROP POLICY ouc_backfill_units_select ON org_units;
    DROP POLICY ouc_backfill_tenants ON tenants;
    RETURN filled;
END
$$;
-- +goose StatementEnd

SELECT org_unit_closure_rebuild_all();

-- R26.3: no nava_migrator policy outlives the migration.
-- +goose StatementBegin
DO $$
DECLARE
    left_over text;
BEGIN
    SELECT string_agg(format('%s.%I', polrelid::regclass, polname), ', ') INTO left_over
    FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles);
    IF left_over IS NOT NULL THEN
        RAISE EXCEPTION 'org_unit_closure backfill: nava_migrator policies left over: %', left_over;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down

DROP FUNCTION IF EXISTS org_unit_closure_rebuild_all();
DROP TABLE IF EXISTS org_unit_closure;
