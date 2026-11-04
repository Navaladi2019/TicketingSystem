-- ADR-004-E5 step (i), expand (TICK-534; D1, D2, D5, D10, D11, I1, I5). Group
-- sync stops deleting a group that Keycloak no longer has and tombstones it
-- instead: deleted_at NULL = in the directory, set = removed at that time. A
-- returning group is revived on the same platform id (D1).
--
-- D11: the database refuses members in a tombstoned group, for every role,
-- the owner included. groups.live is generated from deleted_at, and
-- group_members (tenant_id, group_id, live) references groups (tenant_id, id,
-- live) with NO ACTION, while CHECK (live) pins the member side to true. So
-- inserting a member into a tombstoned group, and tombstoning a group that
-- still has a member, both fail with 23503. The existing ON DELETE CASCADE
-- FK (group_members_group_fkey) stays: until step (ii) group sync may still
-- DELETE a group, and the cascade removes its members before the NO ACTION
-- check runs at the end of the statement.
--
-- D5: when the recorded admins or auditors group is tombstoned, sync deletes
-- its pointer row in the same transaction. nava_groupsync may delete a
-- pointer only when its group is tombstoned (the policies below), and only in
-- its own tenant. The subquery has no deleted_at IS NULL filter on purpose
-- (D4), and the outer columns are qualified with the pointer table (Security
-- N1): unqualified, group_id would silently resolve inside the subquery as
-- soon as groups gains a column of that name. I1: the DELETE ... WHERE and
-- the "was the recorded group removed" probe read tenant_auditor_group, so
-- nava_groupsync joins tenant_auditor_group_select's TO list here (its
-- SELECT (tenant_id, group_id) grant is in db/grants.yaml). The pointer FKs
-- stay ON DELETE CASCADE: the pointer tables are not audited (R29).
--
-- Grants (db/grants.yaml): UPDATE (path, deleted_at) on groups, never id or
-- keycloak_group_id; DELETE on both pointer tables; SELECT (tenant_id,
-- group_id) on tenant_auditor_group. DELETE on groups is revoked in step
-- (ii), TICK-538, not here.
--
-- Locks (I5), measured on PostgreSQL 16.10 with 50,000 groups and 2,500,000
-- memberships (scratch/e5-groups-tombstone/): adding the stored generated
-- column rewrites groups under ACCESS EXCLUSIVE, 205 ms; group_members gets
-- a constant default (no rewrite), 1.7 ms. The CHECK and the FK are NOT VALID
-- here and validated by the next migration under SHARE UPDATE EXCLUSIVE.
-- lock_timeout keeps this migration from queueing reads behind a long sync
-- transaction: it fails and is re-run instead.
--
-- Index: UNIQUE (tenant_id, id, live) is what the FK must reference (2.9 MB at
-- 50,000 groups). The membership side needs none: group_members_pkey leads
-- with (tenant_id, group_id), which serves the NO ACTION check.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE groups
    ADD COLUMN deleted_at timestamptz,
    ADD COLUMN live boolean GENERATED ALWAYS AS (deleted_at IS NULL) STORED,
    ADD CONSTRAINT groups_live_key UNIQUE (tenant_id, id, live);

ALTER TABLE group_members
    ADD COLUMN live boolean NOT NULL DEFAULT true,
    ADD CONSTRAINT group_members_live_check CHECK (live) NOT VALID,
    ADD CONSTRAINT group_members_group_live_fkey FOREIGN KEY (tenant_id, group_id, live)
        REFERENCES groups (tenant_id, id, live) NOT VALID;

CREATE POLICY tenant_admin_group_delete_tombstoned ON tenant_admin_group FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM groups g
                        WHERE g.tenant_id = tenant_admin_group.tenant_id
                          AND g.id = tenant_admin_group.group_id
                          AND g.deleted_at IS NOT NULL));

CREATE POLICY tenant_auditor_group_delete_tombstoned ON tenant_auditor_group FOR DELETE TO nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND EXISTS (SELECT 1 FROM groups g
                        WHERE g.tenant_id = tenant_auditor_group.tenant_id
                          AND g.id = tenant_auditor_group.group_id
                          AND g.deleted_at IS NOT NULL));

ALTER POLICY tenant_auditor_group_select ON tenant_auditor_group TO nava_app, nava_worker, nava_groupsync;
