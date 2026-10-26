-- T252a (TICK-254): the admins a tenant has invited who have not signed in
-- yet (TICK-252 option 2, CTO 2026-10-27). DirectoryAdmin.InviteUser writes
-- a row instead of creating a Keycloak user; after the invitee's first broker
-- login, group sync places them in /t/<tenant_id>/ and consumes the row.
--
-- One index, the primary key (tenant_id, email): it is the UNIQUE (tenant_id,
-- email) Security requires, the re-invite upsert's conflict target, and the
-- consume UPDATE's lookup. No other query path exists.
--
-- nava_onboard (through internal/onboard) invites: it inserts and re-invites
-- with INSERT ... ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at =
-- EXCLUDED.expires_at WHERE pending_invite.consumed_at IS NULL. ON CONFLICT
-- DO UPDATE runs the UPDATE policy on the existing row, so nava_onboard is on
-- it; its UPDATE grant is expires_at alone, so it can never clear consumed_at
-- and a consumed invite stays consumed. nava_groupsync consumes: UPDATE ...
-- SET consumed_at = now() WHERE ... consumed_at IS NULL AND expires_at >
-- now() RETURNING, in its tenant transaction (TICK-252 condition 2d). No role
-- may DELETE: a row is the record that an invite was issued and used.
-- Grants: db/grants.yaml.
--
-- Empty new table, so no lock worth planning for.
-- Design and EXPLAIN evidence: docs/data/identity.md.

-- +goose Up

-- email is as keycloak.normalizeEmail returns it: trimmed, lower case. The
-- CHECK keeps a row the normalized lookup could never match from being
-- written. consumed_at NULL = not used yet.
CREATE TABLE pending_invite (
    tenant_id   uuid        NOT NULL,
    email       text        NOT NULL,
    expires_at  timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pending_invite_pkey PRIMARY KEY (tenant_id, email),
    CONSTRAINT pending_invite_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id),
    CONSTRAINT pending_invite_email_check CHECK (email = lower(btrim(email)) AND strpos(email, '@') > 0),
    CONSTRAINT pending_invite_expires_check CHECK (expires_at > created_at)
);
ALTER TABLE pending_invite ENABLE ROW LEVEL SECURITY;
ALTER TABLE pending_invite FORCE ROW LEVEL SECURITY;

CREATE POLICY pending_invite_select ON pending_invite FOR SELECT TO nava_onboard, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY pending_invite_insert ON pending_invite FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY pending_invite_update ON pending_invite FOR UPDATE TO nava_onboard, nava_groupsync
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
