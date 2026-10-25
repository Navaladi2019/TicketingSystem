-- T11 isolation suite seed, run by the bootstrap superuser (RLS does not
-- apply to it) after db-roles and every migration. Tenant A and tenant B,
-- with tenant B holding more rows than tenant A in every table.
--
-- Every tenant table needs rows of both tenants: the suite fails a table
-- that has none, since 0 rows from an empty table proves nothing. A
-- migration that adds a tenant table adds its rows here.
--
-- Nothing is expired, so the purge policies reach no row; dedupe and
-- idempotency keys repeat across tenants on purpose (registry.go).

INSERT INTO tenants (id, state) VALUES
    ('0a000000-0000-4000-8000-00000000000a', 'active'),
    ('0b000000-0000-4000-8000-00000000000b', 'active');

INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES
    ('0a000000-0000-4000-8000-00000000000a', 'https://idp.example/realms/nava', 'org-a', '{}'),
    ('0b000000-0000-4000-8000-00000000000b', 'https://idp.example/realms/nava', 'org-b', '{idp-b}'),
    ('0b000000-0000-4000-8000-00000000000b', 'https://idp.example/realms/nava', 'org-b2', '{}');

INSERT INTO records (tenant_id, id) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000000a1'),
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000000a2'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000000b1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000000b2'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000000b3');

INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq) VALUES
    ('0a000000-0000-4000-8000-0000000001a1', '0a000000-0000-4000-8000-00000000000a', 'record.changed', 'shared-dedupe-key', 'r-a1', 1),
    ('0b000000-0000-4000-8000-0000000001b1', '0b000000-0000-4000-8000-00000000000b', 'record.changed', 'shared-dedupe-key', 'r-b1', 1),
    ('0b000000-0000-4000-8000-0000000001b2', '0b000000-0000-4000-8000-00000000000b', 'record.changed', 'b-only', 'r-b2', 1);

INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000001a1', '0a000000-0000-4000-8000-0000000000a1', 'record.changed'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000001b1', '0b000000-0000-4000-8000-0000000000b1', 'record.changed'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000001b2', '0b000000-0000-4000-8000-0000000000b2', 'record.changed');

INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000002a1', 'shared-idempotency-key', '\x00', now() + interval '1 day'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000002b1', 'shared-idempotency-key', '\x00', now() + interval '1 day'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000002b1', 'b-only', '\x00', now() + interval '1 day');

-- Identity tables (T16). The Keycloak ids repeat across tenants on purpose,
-- like the dedupe keys: a lookup by one alone would find both tenants' rows.
INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000003a1', 'kc-user-shared', 'a1@tenant-a.test', true, '[]', now()),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000003b1', 'kc-user-shared', 'b1@tenant-b.test', true,
        '[{"identityProvider": "idp-b", "userId": "b1", "userName": "b1"}]', now()),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000003b2', 'kc-user-b2', NULL, true, NULL, now());

INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a1', 'kc-group-shared', '/t/0a000000-0000-4000-8000-00000000000a/agents'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b1', 'kc-group-shared', '/t/0b000000-0000-4000-8000-00000000000b/agents'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b2', 'kc-group-b2', '/t/0b000000-0000-4000-8000-00000000000b/admins');

INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a1', '0a000000-0000-4000-8000-0000000003a1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b1', '0b000000-0000-4000-8000-0000000003b1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b2', '0b000000-0000-4000-8000-0000000003b2');

INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000005a1', NULL, 'kc-ou-shared', 'Root A'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000005b1', NULL, 'kc-ou-shared', 'Root B'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000005b2', '0b000000-0000-4000-8000-0000000005b1', 'kc-ou-b2', 'Support B');
