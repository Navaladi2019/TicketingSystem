-- TICK-782 (S11b, blocks TICK-712): the worker writes records through the
-- records pipeline as nava_worker. The update path reads the record through
-- records/store's recordSelect, whose
--   LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
-- reads g.deleted_at (the assignment group's tombstone). nava_worker had no
-- grant and no policy on groups, so every update_record step got 42501.
--
-- groups_select gains nava_worker, under its existing tenant predicate: the
-- worker sees its own transaction's tenant's groups, as nava_app does. The
-- grant (db/grants.yaml) is SELECT (tenant_id, id, deleted_at) only, the
-- columns the join and the policy read: never path or keycloak_group_id.
-- No INSERT, UPDATE or DELETE.
--
-- ALTER POLICY ... TO is catalog-only: no rewrite, a brief ACCESS EXCLUSIVE
-- lock on groups (lock_timeout applies), no index. groups_pkey
-- (tenant_id, id) backs both the join and the tenant predicate.

-- +goose Up

ALTER POLICY groups_select ON groups TO nava_app, nava_groupsync, nava_worker;

-- +goose Down

ALTER POLICY groups_select ON groups TO nava_app, nava_groupsync;
