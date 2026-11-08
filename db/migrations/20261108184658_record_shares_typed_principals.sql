-- P3-30a (TICK-625, closes AR-S10-01): a record share names a principal of
-- the right kind that exists in the share's tenant.
--
-- record_shares.principal_id had no foreign key: one column cannot reference
-- both principals and groups. It is replaced by two written columns:
--
--   user_id   set for a 'user' share, FK (tenant_id, user_id) -> principals
--   group_id  set for a 'group' share, FK (tenant_id, group_id) -> groups
--
-- record_shares_principal_check holds exactly one of them set, the one
-- principal_kind names. It replaces record_shares_principal_kind_check, which
-- it implies. Both foreign keys are composite, as every key here is (D-3-2):
-- PostgreSQL checks them as the owner, past RLS, so tenant A cannot name
-- tenant B's user or group (23503). A group id written as a user, a user id
-- written as a group, or an id the tenant does not have, fails the foreign
-- key (23503).
--
-- principal_id stays, as a stored generated column,
-- COALESCE(user_id, group_id), so the primary key
-- (tenant_id, record_id, principal_kind, principal_id) and the list join's
-- record_shares_principal (tenant_id, principal_id) keep their shape and the
-- access pattern P3-34 was planned on (docs/data/acl-tables.md). No writer
-- can set it: an INSERT naming principal_id is refused (428C9).
--
-- No index backs the two new foreign keys from the parent side. Their only
-- cost is a delete of a principals or groups row, and neither is deleted:
-- principals and groups are tombstoned (00010, ADR-004-E5), and no role may
-- DELETE a group. The same holds for records_assignment_group_fkey
-- (AR-TICK-538-01).
--
-- Online: record_shares is empty in every database. The table is new in this
-- story (20261108124249, TICK-615), main has never shipped it, and no code
-- path writes it yet. So the column swap, the rewrite the generated column
-- costs and the plain index builds touch no rows, and the ACCESS EXCLUSIVE
-- lock is held for catalogue work only, under lock_timeout. A database that
-- did hold a share fails the primary key (23502, principal_id is NULL for an
-- old row), and the whole migration rolls back with nothing changed.
--
-- Down keeps the generated values: DROP EXPRESSION turns principal_id back
-- into a written column, then user_id and group_id go.

-- +goose Up

SET LOCAL lock_timeout = '5s';
ALTER TABLE record_shares
    DROP CONSTRAINT record_shares_principal_kind_check,
    DROP COLUMN principal_id,
    ADD COLUMN user_id uuid,
    ADD COLUMN group_id uuid,
    ADD COLUMN principal_id uuid GENERATED ALWAYS AS (COALESCE(user_id, group_id)) STORED,
    ADD CONSTRAINT record_shares_pkey PRIMARY KEY (tenant_id, record_id, principal_kind, principal_id),
    ADD CONSTRAINT record_shares_user_fkey FOREIGN KEY (tenant_id, user_id)
        REFERENCES principals (tenant_id, id),
    ADD CONSTRAINT record_shares_group_fkey FOREIGN KEY (tenant_id, group_id)
        REFERENCES groups (tenant_id, id),
    ADD CONSTRAINT record_shares_principal_check CHECK (
        (principal_kind = 'user' AND user_id IS NOT NULL AND group_id IS NULL)
        OR (principal_kind = 'group' AND group_id IS NOT NULL AND user_id IS NULL));
CREATE INDEX record_shares_principal ON record_shares (tenant_id, principal_id);

-- +goose Down

SET LOCAL lock_timeout = '5s';
ALTER TABLE record_shares
    DROP CONSTRAINT record_shares_principal_check,
    DROP CONSTRAINT record_shares_group_fkey,
    DROP CONSTRAINT record_shares_user_fkey,
    ALTER COLUMN principal_id DROP EXPRESSION,
    DROP COLUMN group_id,
    DROP COLUMN user_id,
    ADD CONSTRAINT record_shares_principal_kind_check CHECK (principal_kind IN ('user', 'group'));
