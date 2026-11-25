-- TICK-947 (S31 P11-14; ADR-010 D7; PLAN-011 C5): org units are tombstoned,
-- never deleted, and gain a membership table. DDL, keys and grants are SP-2's
-- (docs/data/acl-format2-caps.md section 5). Platform DDL, no per-tenant DDL.
--
-- 1. org_units.deleted_at: NULL = live, set = removed from the directory at
--    that time. live is generated from it and UNIQUE (tenant_id, id, live) is
--    what org_unit_members' live FK references, as groups does (ADR-004-E5
--    D11). org_units_delete is dropped and nava_groupsync loses DELETE
--    (db/grants.yaml): a unit that leaves Keycloak is tombstoned, so
--    records.org_unit_id and stored under ids never dangle.
--
-- 2. org_unit_members (tenant_id, org_unit_id, principal_id, is_primary).
--    Written by nava_groupsync only; read by nava_app and nava_worker. FORCE
--    RLS with the app.tenant_id policies org_units has. Both FKs carry
--    tenant_id. The live FK (NO ACTION) with CHECK (live) refuses a member of
--    a tombstoned unit, and refuses tombstoning a unit that still has a
--    member, for every role, with 23503. A principal's rows go with the
--    principal (ON DELETE CASCADE), as group_members does.
--
-- 3. Indexes (3): the primary key leads with (tenant_id, org_unit_id) and
--    backs the NO ACTION check when a unit is tombstoned and groupsync's
--    per-unit reads; org_unit_members_principal serves "units of a
--    principal" (the set S seed, D7) as an index-only scan; the partial
--    unique index is D-11-6's one primary unit per (tenant, principal).
--
-- 4. org_units_select gains nava_worker (its column grant is
--    tenant_id, id, parent_id, deleted_at): the worker resolves under for
--    workflow policy tests.
--
-- Locks: adding the stored generated column rewrites org_units under ACCESS
-- EXCLUSIVE. org_units holds one row per Keycloak org-unit group; groups took
-- 205 ms at 50,000 rows for the same change (scratch/e5-groups-tombstone/).
-- The new table is empty, so its constraints are added valid. lock_timeout
-- makes the migration fail and be re-run instead of queueing reads behind a
-- long sync transaction.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE org_units
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED,
    ADD CONSTRAINT org_units_live_key UNIQUE (tenant_id, id, live);

DROP POLICY org_units_delete ON org_units;
ALTER POLICY org_units_select ON org_units TO nava_app, nava_worker, nava_groupsync;

CREATE TABLE org_unit_members (
    tenant_id    uuid    NOT NULL,
    org_unit_id  uuid    NOT NULL,
    principal_id uuid    NOT NULL,
    is_primary   boolean NOT NULL DEFAULT false,
    live         boolean NOT NULL DEFAULT true,
    CONSTRAINT org_unit_members_pkey PRIMARY KEY (tenant_id, org_unit_id, principal_id),
    CONSTRAINT org_unit_members_live_check CHECK (live),
    CONSTRAINT org_unit_members_unit_live_fkey FOREIGN KEY (tenant_id, org_unit_id, live)
        REFERENCES org_units (tenant_id, id, live),
    CONSTRAINT org_unit_members_principal_fkey FOREIGN KEY (tenant_id, principal_id)
        REFERENCES principals (tenant_id, id) ON DELETE CASCADE
);
CREATE INDEX org_unit_members_principal ON org_unit_members (tenant_id, principal_id, org_unit_id);
CREATE UNIQUE INDEX org_unit_members_one_primary ON org_unit_members (tenant_id, principal_id) WHERE is_primary;
ALTER TABLE org_unit_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE org_unit_members FORCE ROW LEVEL SECURITY;

CREATE POLICY org_unit_members_select ON org_unit_members FOR SELECT TO nava_app, nava_worker, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_unit_members_insert ON org_unit_members FOR INSERT TO nava_groupsync
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_unit_members_update ON org_unit_members FOR UPDATE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY org_unit_members_delete ON org_unit_members FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose Down

-- Safe only before any unit is tombstoned: a tombstoned row would come back
-- as a live unit. Run TICK-948's sync rollback first.
SET LOCAL lock_timeout = '5s';

DROP TABLE org_unit_members;
ALTER POLICY org_units_select ON org_units TO nava_app, nava_groupsync;
CREATE POLICY org_units_delete ON org_units FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
ALTER TABLE org_units
    DROP CONSTRAINT org_units_live_key,
    DROP COLUMN live,
    DROP COLUMN deleted_at;
