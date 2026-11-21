-- TICK-1025 (S32, ADR-010 D9; blocks TICK-957): a workflow condition step's
-- named-policy test resolves the record's requester or assignee inside the
-- run's tenant transaction, as nava_worker, through authz.PrincipalOf
-- (internal/authz/explain.go, targetSQL):
--   SELECT p.id, ARRAY(SELECT gm.group_id FROM group_members gm
--                      JOIN groups g ON ... AND g.deleted_at IS NULL ...)
--   FROM principals p WHERE p.tenant_id = $1 AND p.id = $2 AND p.deleted_at IS NULL
-- nava_worker had no grant and no policy on principals or group_members, so
-- the activity got 42501 and would retry forever.
--
-- principals_select and group_members_select gain nava_worker, under their
-- existing canonical tenant predicate: the worker sees only its
-- transaction's tenant's rows, as nava_app does, and with no tenant or an
-- empty one, none. The grants (db/grants.yaml) are column-limited to what
-- targetSQL and the policies read:
--   principals    (tenant_id, id, deleted_at): never keycloak_user_id,
--                 email, enabled, federated_links or synced_at;
--   group_members (tenant_id, group_id, principal_id).
-- groups (tenant_id, id, deleted_at) is already granted (TICK-782,
-- 20261113190000). No INSERT, UPDATE or DELETE.
--
-- ALTER POLICY ... TO is catalog-only and idempotent: no scan, no rewrite, no
-- index; a brief ACCESS EXCLUSIVE lock on each table that lock_timeout
-- bounds. principals_pkey (tenant_id, id) backs the principal probe and its
-- tenant predicate; group_members_principal (tenant_id, principal_id) backs
-- the membership lookup and its tenant predicate.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER POLICY principals_select ON principals TO nava_app, nava_groupsync, nava_worker;
ALTER POLICY group_members_select ON group_members TO nava_app, nava_groupsync, nava_worker;

-- +goose Down

SET LOCAL lock_timeout = '5s';

ALTER POLICY principals_select ON principals TO nava_app, nava_groupsync;
ALTER POLICY group_members_select ON group_members TO nava_app, nava_groupsync;
