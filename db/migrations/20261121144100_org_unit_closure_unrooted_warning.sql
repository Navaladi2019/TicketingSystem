-- TICK-1058 (S31 follow-up; the Security & Multi-Tenancy Reviewer's attack-4
-- decision on TICK-1034): org_unit_closure_rebuild_all() reports, per tenant,
-- the live org units that have no (u, u) row after the rebuild. Since
-- TICK-1034 such a unit (an orphan, or a unit on a parent cycle) is outside
-- every region. The report is a WARNING, never a failure: the function is a
-- backfill and repair tool, and failing it would leave every other tenant's
-- closure stale too. Group sync's rebuild reports the same count as the
-- groupsync_org_closure_unrooted_units metric (internal/groupsync/closure.go).
--
-- The body is 20261121050000's with the count added after each tenant's
-- INSERT; a shipped migration is frozen, so the change is a replace.
-- CREATE OR REPLACE takes no table lock and keeps the signature and grants.
-- Down restores the 20261121050000 body.

-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION org_unit_closure_rebuild_all() RETURNS bigint
    LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path = pg_catalog, public AS $$
DECLARE
    tid uuid;
    filled bigint := 0;
    n bigint;
    unrooted bigint;
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
        -- TICK-1058: live units left without a self row (an orphan, or a
        -- unit on a parent cycle) are outside every region. One anti-join,
        -- each unit probing org_unit_closure_pkey.
        SELECT count(*) INTO unrooted FROM org_units u
        WHERE u.tenant_id = tid AND u.deleted_at IS NULL
          AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
              WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id);
        IF unrooted > 0 THEN
            RAISE WARNING 'org_unit_closure_rebuild_all: tenant % has % live org units with no closure self row; they are outside every region', tid, unrooted
                USING ERRCODE = '01000';
        END IF;
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

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION org_unit_closure_rebuild_all() RETURNS bigint
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
