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

-- The cascade start key (P3-12, ADR-001 A5): tenant B also holds record id
-- a1, tenant A's record, both rows under the same causation origin 0d…9d1,
-- and only tenant B has claimed definition 0d…9d2 on it, so a key read or
-- claimed without its tenant would meet tenant B's row. Tenant A keeps its
-- two records (the CI-2 harness counts them); its own claim on a1 is for
-- another definition, 0d…9d3.
UPDATE records SET causation_origin_id = '0d000000-0000-4000-8000-0000000009d1', causation_depth = 0
WHERE tenant_id = '0a000000-0000-4000-8000-00000000000a' AND id = '0a000000-0000-4000-8000-0000000000a1';
INSERT INTO records (tenant_id, id, causation_origin_id, causation_depth) VALUES
    ('0b000000-0000-4000-8000-00000000000b', '0a000000-0000-4000-8000-0000000000a1', '0d000000-0000-4000-8000-0000000009d1', 0);

INSERT INTO cascade_start (tenant_id, causation_origin_id, record_id, definition_id, outbox_id) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0d000000-0000-4000-8000-0000000009d1', '0a000000-0000-4000-8000-0000000000a1',
        '0d000000-0000-4000-8000-0000000009d3', '0a000000-0000-4000-8000-0000000009a1'),
    ('0b000000-0000-4000-8000-00000000000b', '0d000000-0000-4000-8000-0000000009d1', '0a000000-0000-4000-8000-0000000000a1',
        '0d000000-0000-4000-8000-0000000009d2', '0b000000-0000-4000-8000-0000000009b1');

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
    -- CI-5.10: a tenant A principal federated to tenant B's IdP, stored as
    -- the alias-only marker group sync writes (TICK-263): no foreign userId.
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000003a2', 'kc-user-a-foreign', NULL, true,
        '[{"identityProvider": "idp-b"}]', now()),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000003b2', 'kc-user-b2', NULL, true, NULL, now());

INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a1', 'kc-group-shared', '/t/0a000000-0000-4000-8000-00000000000a/agents'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b1', 'kc-group-shared', '/t/0b000000-0000-4000-8000-00000000000b/agents'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b2', 'kc-group-b2', '/t/0b000000-0000-4000-8000-00000000000b/admins'),
    -- ADR-002 AC2: a B-only marker no tenant A response may ever contain, and
    -- a second tenant A group that a1 is not in.
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b3', 'kc-group-b-marker', '/t/0b000000-0000-4000-8000-00000000000b/b-only-marker'),
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a2', 'kc-group-a2', '/t/0a000000-0000-4000-8000-00000000000a/a-others');

-- P3-04: each seed tenant's admins group (a seeded group of the same tenant).
INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b2');

INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a1', '0a000000-0000-4000-8000-0000000003a1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b1', '0b000000-0000-4000-8000-0000000003b1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b2', '0b000000-0000-4000-8000-0000000003b2'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000004b3', '0b000000-0000-4000-8000-0000000003b1'),
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000004a2', '0a000000-0000-4000-8000-0000000003a2');

INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000005a1', NULL, 'kc-ou-shared', 'Root A'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000005b1', NULL, 'kc-ou-shared', 'Root B'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000005b2', '0b000000-0000-4000-8000-0000000005b1', 'kc-ou-b2', 'Support B');

-- Email domains (T18a). A domain is unique realm-wide, so they cannot repeat.
INSERT INTO tenant_email_domain (tenant_id, domain) VALUES
    ('0a000000-0000-4000-8000-00000000000a', 'tenant-a.test'),
    ('0b000000-0000-4000-8000-00000000000b', 'tenant-b.test'),
    ('0b000000-0000-4000-8000-00000000000b', 'b2.tenant-b.test');

-- Pending invites (T252a). The same email is invited by both tenants, so a
-- lookup by email alone would find both rows without isolation; B also has a
-- consumed invite.
INSERT INTO pending_invite (tenant_id, email, expires_at, consumed_at) VALUES
    ('0a000000-0000-4000-8000-00000000000a', 'shared-invite@example.test', now() + interval '7 days', NULL),
    ('0b000000-0000-4000-8000-00000000000b', 'shared-invite@example.test', now() + interval '7 days', NULL),
    ('0b000000-0000-4000-8000-00000000000b', 'used-invite@example.test', now() + interval '7 days', now());

-- Entity types and versions (P3-02, TICK-340). The key and number_prefix
-- repeat across tenants on purpose: a lookup by either alone would find both
-- tenants' rows without isolation. schema_sha256 is computed from the stored
-- text, as the CHECK requires. fixed_columns and states are JSON objects, as
-- the API writes them: the store scans them into a map and a StateModel, so
-- an array turns every read of the type into a 500 (TICK-378).
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000006a1', 'ticket', 'Ticket A', 'TCK', 1),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000006b1', 'ticket', 'Ticket B', 'TCK', 2),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000006b2', 'incident', 'Incident B', 'INC', 1);

INSERT INTO entity_definition_versions (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
SELECT v.tenant_id::uuid, v.id::uuid, v.definition_id::uuid, v.version, v.schema::json,
       sha256(convert_to(v.schema, 'UTF8')), '{}',
       '{"initial": "new", "states": ["new"], "transitions": []}', v.compat, v.published_by::uuid
FROM (VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000007a1', '0a000000-0000-4000-8000-0000000006a1', 1,
        '{"type": "object"}', 'initial', '0a000000-0000-4000-8000-0000000003a1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000007b1', '0b000000-0000-4000-8000-0000000006b1', 1,
        '{"type": "object"}', 'initial', '0b000000-0000-4000-8000-0000000003b1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000007b2', '0b000000-0000-4000-8000-0000000006b1', 2,
        '{"type": "object", "title": "B v2"}', 'compatible', '0b000000-0000-4000-8000-0000000003b1'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000007b3', '0b000000-0000-4000-8000-0000000006b2', 1,
        '{"type": "object"}', 'initial', NULL)
) AS v (tenant_id, id, definition_id, version, schema, compat, published_by);

-- Records pipeline (P3-09, TICK-353). Tenant B's ticket record carries
-- pipeline columns naming its own principal and group, with the same number
-- as tenant A's, so a lookup by number alone would find both tenants' rows;
-- the idempotency key repeats across tenants for the same reason. Tenant A's
-- records stay bare PLAN-001 rows, which the all-or-none CHECK allows.
UPDATE records SET entity_definition_id = '0a000000-0000-4000-8000-0000000006a1', schema_version = 1,
       number = 'TCK0000001', state = 'new', requester_id = '0a000000-0000-4000-8000-0000000003a1',
       created_by = '0a000000-0000-4000-8000-0000000003a1', assignment_group_id = '0a000000-0000-4000-8000-0000000004a1'
WHERE tenant_id = '0a000000-0000-4000-8000-00000000000a' AND id = '0a000000-0000-4000-8000-0000000000a2';
UPDATE records SET entity_definition_id = '0b000000-0000-4000-8000-0000000006b1', schema_version = 2,
       number = 'TCK0000001', state = 'new', requester_id = '0b000000-0000-4000-8000-0000000003b1',
       created_by = '0b000000-0000-4000-8000-0000000003b1', assignment_group_id = '0b000000-0000-4000-8000-0000000004b1'
WHERE tenant_id = '0b000000-0000-4000-8000-00000000000b' AND id = '0b000000-0000-4000-8000-0000000000b2';

INSERT INTO audit_log (tenant_id, id, record_id, action, actor_id, source, after, idempotency_key) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000008a1', '0a000000-0000-4000-8000-0000000000a2',
        'create', '0a000000-0000-4000-8000-0000000003a1', 'user', '{}', 'shared-audit-key'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000008b1', '0b000000-0000-4000-8000-0000000000b2',
        'create', '0b000000-0000-4000-8000-0000000003b1', 'user', '{}', 'shared-audit-key'),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000008b2', NULL,
        'cascade_rejected', NULL, 'workflow', NULL, NULL);

INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES
    ('0a000000-0000-4000-8000-00000000000a', '0a000000-0000-4000-8000-0000000006a1', 2),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000006b1', 2),
    ('0b000000-0000-4000-8000-00000000000b', '0b000000-0000-4000-8000-0000000006b2', 5);
