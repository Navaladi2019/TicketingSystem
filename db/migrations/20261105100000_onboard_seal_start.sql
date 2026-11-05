-- P5-33 (TICK-441, ADR-001-A6.2, ADR-004 R16): onboarding seeds a tenant's
-- audit_chain_head row and its first audit.seal start row. nava_onboard gets
-- INSERT on audit_chain_head and on outbox, behind pinned provisioning
-- policies (the E3 pattern): the transaction's own tenant, only while it is
-- provisioning, and on outbox only kind = 'audit.seal'. It has no SELECT on
-- outbox and none on audit_chain_head, so it can neither read nor update a
-- row; onboard.PGStore.SeedAuditChain is a plain INSERT, no RETURNING, and
-- idempotent through the head's primary key (ON CONFLICT DO NOTHING).
--
-- Policies only: a catalogue update, no rewrite, no index. Grants:
-- db/grants.yaml. The policies are registered in
-- internal/db/catalog.PinnedTenantPolicies.

-- +goose Up

CREATE POLICY audit_chain_head_onboard_insert ON audit_chain_head FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'));

CREATE POLICY outbox_onboard_seal_insert ON outbox FOR INSERT TO nava_onboard
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
                AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning')
                AND kind = 'audit.seal');

-- +goose Down

DROP POLICY outbox_onboard_seal_insert ON outbox;
DROP POLICY audit_chain_head_onboard_insert ON audit_chain_head;
