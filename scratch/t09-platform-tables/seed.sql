-- T09 benchmark data. Runs as the bootstrap superuser (bypasses RLS), after
-- nava-db roles + migrate built the real schema, roles and grants.
--
-- 200 tenants with skew: tenant 1 holds 40% of every table, 199 small
-- tenants share the rest.
--   records          1,000,000
--   outbox           2,000,000 dispatched + 50,000 pending
--   workflow_ref     1,000,000 (90% closed)
--   idempotency_keys 1,000,000 (10% expired)
\set ON_ERROR_STOP 1
\timing on

CREATE TEMP TABLE t AS
SELECT n, md5('tenant' || n)::uuid AS id FROM generate_series(1, 200) n;

INSERT INTO tenants (id, state) SELECT id, 'active' FROM t;
INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases)
SELECT id, 'https://kc.example/realms/nava', 'org-' || n, '{corp-saml}' FROM t;

-- Tenant of row g: 40% to tenant 1, the rest round-robin over 2..200.
CREATE TEMP TABLE tenant_of AS
SELECT g, (SELECT id FROM t WHERE n = CASE WHEN g % 5 < 2 THEN 1 ELSE 2 + g % 199 END) AS tenant_id
FROM generate_series(1, 2050000) g;
CREATE INDEX ON tenant_of (g);
ANALYZE tenant_of;

INSERT INTO records (tenant_id, id, version)
SELECT tenant_id, md5('rec' || g)::uuid, g % 7 FROM tenant_of WHERE g <= 1000000;

INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload,
                    created_at, available_at, attempts, dispatched_at)
SELECT gen_random_uuid(), tenant_id, 'workflow.signal', 'd' || g, 'rec:' || md5('rec' || g), g,
       jsonb_build_object('record_id', md5('rec' || g)::uuid),
       now() - interval '20 days' + g * interval '0.8 s', now() - interval '20 days' + g * interval '0.8 s', 1,
       now() - interval '20 days' + g * interval '0.8 s' + interval '1 s'
FROM tenant_of WHERE g <= 2000000;
INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq, payload, created_at, available_at)
SELECT gen_random_uuid(), tenant_id, 'workflow.signal', 'd' || g, 'rec:' || md5('rec' || (g - 2000000)), g,
       jsonb_build_object('record_id', md5('rec' || (g - 2000000))::uuid),
       now() - interval '10 min' + (g - 2000000) * interval '10 ms', now() - interval '10 min' + (g - 2000000) * interval '10 ms'
FROM tenant_of WHERE g > 2000000;

INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, created_at, closed_at)
SELECT tenant_id, gen_random_uuid(), md5('rec' || g)::uuid, 'approval',
       now() - interval '20 days' + g * interval '1.5 s',
       CASE WHEN g % 10 <> 0 THEN now() - interval '20 days' + g * interval '1.5 s' + interval '1 hour' END
FROM tenant_of WHERE g <= 1000000;

INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, state, response_status, response_body, created_at, expires_at)
SELECT tenant_id, md5('principal' || g % 5000)::uuid, 'k' || g, sha256(('req' || g)::bytea), 'completed', 201,
       convert_to('{"id":"' || md5('rec' || g) || '"}', 'UTF8'),
       now() - interval '1 day' + g * interval '0.08 s',
       CASE WHEN g % 10 = 0 THEN now() - interval '1 hour' ELSE now() + interval '1 day' END
FROM tenant_of WHERE g <= 1000000;

VACUUM ANALYZE tenants, tenant_identity_binding, records, outbox, workflow_ref, idempotency_keys;
