-- T09 (TICK-64): the platform tables of ADR-001 A.3, each under ENABLE + FORCE
-- ROW LEVEL SECURITY. Runs as nava_migrator; grants come from db/grants.yaml.
--
-- Every policy tests app.tenant_id, through
--     NULLIF(current_setting('app.tenant_id', true), '')::uuid
-- (no tenant set, or an empty one, matches no row), except the 10 on the A.3
-- allow-list, each FOR one command and TO named roles:
--     outbox_relay_select, outbox_relay_update, binding_lookup, binding_onboard,
--     idempotency_purge, idempotency_purge_select, outbox_purge,
--     outbox_purge_select, workflow_ref_purge, workflow_ref_purge_select.
-- TestPolicyAllowList keeps it that way. A new policy that does not test
-- app.tenant_id is an ADR change, not a migration.
--
-- Empty new tables, so plain CREATE INDEX holds no lock worth planning for.

-- +goose Up

-- tenants is keyed on id (PLAN-001 D-P16). nava_onboard creates a row in
-- 'provisioning' inside InTenantTx(newTenantID), seeds it, then activates it.
-- onboarding_request_hash is the operator idempotency record (T21, D-P11).
CREATE TABLE tenants (
    id                      uuid        NOT NULL,
    state                   text        NOT NULL,
    onboarding_request_hash bytea,
    created_at              timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenants_pkey PRIMARY KEY (id),
    CONSTRAINT tenants_state_check CHECK (state IN ('provisioning', 'active'))
);
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;

-- The API reads its own tenant's row (for example its state) and nothing else.
CREATE POLICY tenants_tenant ON tenants FOR SELECT TO nava_app
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- Onboarding sees only the tenant of its transaction. binding_onboard's
-- subquery reads tenants through this policy, so it too sees only that tenant.
CREATE POLICY tenants_onboard_select ON tenants FOR SELECT TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY tenants_onboard_insert ON tenants FOR INSERT TO nava_onboard
    WITH CHECK (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND state = 'provisioning');
-- provisioning -> active, own tenant only; nothing moves back.
CREATE POLICY tenants_activate ON tenants FOR UPDATE TO nava_onboard
    USING (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
           AND state = 'provisioning')
    WITH CHECK (id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND state = 'active');

-- Tenant resolution (A.1 step 4) looks a token's (issuer, org_id) up before
-- any tenant is known, so binding_lookup cannot test app.tenant_id.
-- UNIQUE (issuer, org_id) is global on purpose: one organization, one tenant.
CREATE TABLE tenant_identity_binding (
    tenant_id   uuid        NOT NULL,
    issuer      text        NOT NULL,
    org_id      text        NOT NULL,
    idp_aliases text[]      NOT NULL DEFAULT '{}',
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenant_identity_binding_pkey PRIMARY KEY (tenant_id, issuer, org_id),
    CONSTRAINT tenant_identity_binding_issuer_org_key UNIQUE (issuer, org_id),
    CONSTRAINT tenant_identity_binding_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants (id)
);
ALTER TABLE tenant_identity_binding ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_identity_binding FORCE ROW LEVEL SECURITY;

CREATE POLICY binding_lookup ON tenant_identity_binding FOR SELECT TO nava_app, nava_groupsync
    USING (true);
-- Only a provisioning tenant that nava_onboard can see (its own, through
-- tenants_onboard_select) can be bound; never an active one.
CREATE POLICY binding_onboard ON tenant_identity_binding FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));

-- Minimal records (T09); PLAN-002 adds entity_definition_id, data jsonb and
-- the rest additively. version is the row lock Enqueue bumps (T29).
CREATE TABLE records (
    tenant_id uuid   NOT NULL,
    id        uuid   NOT NULL,
    version   bigint NOT NULL DEFAULT 0,
    CONSTRAINT records_pkey PRIMARY KEY (tenant_id, id)
);
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;

CREATE POLICY records_tenant ON records TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- The transactional outbox (A.3). payload holds references only (CI-7).
CREATE TABLE outbox (
    id              uuid        NOT NULL, -- UUIDv7
    tenant_id       uuid        NOT NULL,
    kind            text        NOT NULL,
    kind_version    smallint    NOT NULL DEFAULT 1,
    dedupe_key      text        NOT NULL,
    target_key      text        NOT NULL,
    target_seq      bigint      NOT NULL,
    causation_id    uuid,
    causation_depth smallint    NOT NULL DEFAULT 0,
    payload         jsonb       NOT NULL DEFAULT '{}',
    trace_context   text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    available_at    timestamptz NOT NULL DEFAULT now(),
    attempts        integer     NOT NULL DEFAULT 0,
    last_error      text,
    dispatched_at   timestamptz,
    dead_at         timestamptz,
    CONSTRAINT outbox_pkey PRIMARY KEY (tenant_id, id),
    CONSTRAINT outbox_dedupe_key UNIQUE (tenant_id, dedupe_key),
    CONSTRAINT outbox_causation_depth_check CHECK (causation_depth >= 0),
    CONSTRAINT outbox_attempts_check CHECK (attempts >= 0)
);
CREATE INDEX outbox_pending_available ON outbox (tenant_id, available_at)
    WHERE dispatched_at IS NULL AND dead_at IS NULL;
CREATE INDEX outbox_pending_target ON outbox (tenant_id, target_key, target_seq)
    WHERE dispatched_at IS NULL AND dead_at IS NULL;
ALTER TABLE outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox FORCE ROW LEVEL SECURITY;

CREATE POLICY outbox_tenant ON outbox TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
-- The relay polls every tenant. Its column grants hide dedupe_key and
-- last_error and let it write only the dispatch columns; it updates only
-- pending rows.
CREATE POLICY outbox_relay_select ON outbox FOR SELECT TO nava_relay
    USING (true);
CREATE POLICY outbox_relay_update ON outbox FOR UPDATE TO nava_relay
    USING (dispatched_at IS NULL AND dead_at IS NULL)
    WITH CHECK (true);
-- Paired purge policies, same predicate: purge sees exactly what it may delete.
CREATE POLICY outbox_purge_select ON outbox FOR SELECT TO nava_purge
    USING (dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days');
CREATE POLICY outbox_purge ON outbox FOR DELETE TO nava_purge
    USING (dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days');

-- One row per workflow the relay starts; the final activity writes closed_at.
CREATE TABLE workflow_ref (
    tenant_id  uuid        NOT NULL,
    outbox_id  uuid        NOT NULL,
    record_id  uuid        NOT NULL,
    kind       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    closed_at  timestamptz,
    CONSTRAINT workflow_ref_pkey PRIMARY KEY (tenant_id, outbox_id),
    CONSTRAINT workflow_ref_record_fkey FOREIGN KEY (tenant_id, record_id) REFERENCES records (tenant_id, id)
);
-- Also the index behind workflow_ref_record_fkey.
CREATE INDEX workflow_ref_record ON workflow_ref (tenant_id, record_id, kind, created_at DESC);
ALTER TABLE workflow_ref ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_ref FORCE ROW LEVEL SECURITY;

CREATE POLICY workflow_ref_tenant ON workflow_ref TO nava_app, nava_worker
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY workflow_ref_purge_select ON workflow_ref FOR SELECT TO nava_purge
    USING (closed_at < now() - interval '30 days');
CREATE POLICY workflow_ref_purge ON workflow_ref FOR DELETE TO nava_purge
    USING (closed_at < now() - interval '30 days');

-- API idempotency (T22). response_body is bytes so a replay is byte-identical.
CREATE TABLE idempotency_keys (
    tenant_id       uuid        NOT NULL,
    principal_id    uuid        NOT NULL,
    key             text        NOT NULL,
    request_hash    bytea       NOT NULL,
    state           text        NOT NULL DEFAULT 'in_flight',
    response_status smallint,
    response_body   bytea,
    created_at      timestamptz NOT NULL DEFAULT now(),
    expires_at      timestamptz NOT NULL,
    CONSTRAINT idempotency_keys_pkey PRIMARY KEY (tenant_id, principal_id, key),
    CONSTRAINT idempotency_keys_state_check CHECK (state IN ('in_flight', 'completed')),
    CONSTRAINT idempotency_keys_completed_check CHECK ((state = 'completed') = (response_status IS NOT NULL))
);
ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE idempotency_keys FORCE ROW LEVEL SECURITY;

CREATE POLICY idempotency_tenant ON idempotency_keys TO nava_app
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY idempotency_purge_select ON idempotency_keys FOR SELECT TO nava_purge
    USING (expires_at < now());
CREATE POLICY idempotency_purge ON idempotency_keys FOR DELETE TO nava_purge
    USING (expires_at < now());
