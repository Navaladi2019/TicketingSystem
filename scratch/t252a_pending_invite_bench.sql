-- T252a (TICK-254): pending_invite plans at volume, as the runtime roles
-- with FORCE RLS on. Run with psql as the superuser on a migrated nava
-- database (a dbtest cluster). Results: docs/data/identity.md.
--
-- Volume: 10,000 tenants with 1-3 invites each (20,000 rows), plus one large
-- tenant (0x2711) with 50,000 invites. A third of all rows are consumed.

\timing on
INSERT INTO tenants (id, state)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid, 'active'
FROM generate_series(1, 10001) g
ON CONFLICT DO NOTHING;

-- The superuser bypasses RLS, so it loads the rows directly.
INSERT INTO pending_invite (tenant_id, email, expires_at, consumed_at)
SELECT ('00000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       'u' || i || '@t' || g || '.example', now() + interval '7 days',
       CASE WHEN (g + i) % 3 = 0 THEN now() END
FROM generate_series(1, 10000) g, generate_series(1, 1 + g % 3) i;
INSERT INTO pending_invite (tenant_id, email, expires_at, consumed_at)
SELECT '00000000-0000-4000-8000-000000002711', 'u' || i || '@big.example', now() + interval '7 days',
       CASE WHEN i % 3 = 0 THEN now() END
FROM generate_series(1, 50000) i;
VACUUM ANALYZE pending_invite;
\timing off

SELECT count(*) AS rows, pg_size_pretty(pg_relation_size('pending_invite')) AS heap,
       pg_size_pretty(pg_indexes_size('pending_invite')) AS indexes
FROM pending_invite;

-- 1. Consume (condition 2d), as nava_groupsync, small and large tenant.
SET ROLE nava_groupsync;
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000000005', true);
EXPLAIN (ANALYZE, BUFFERS) UPDATE pending_invite SET consumed_at = now()
WHERE tenant_id = '00000000-0000-4000-8000-000000000005' AND email = 'u1@t5.example'
  AND consumed_at IS NULL AND expires_at > now() RETURNING tenant_id, email;
ROLLBACK;
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000002711', true);
EXPLAIN (ANALYZE, BUFFERS) UPDATE pending_invite SET consumed_at = now()
WHERE tenant_id = '00000000-0000-4000-8000-000000002711' AND email = 'u49999@big.example'
  AND consumed_at IS NULL AND expires_at > now() RETURNING tenant_id, email;
ROLLBACK;
-- 2. Consume aimed at the large tenant from a small tenant's transaction.
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000000005', true);
EXPLAIN (ANALYZE, BUFFERS) UPDATE pending_invite SET consumed_at = now()
WHERE tenant_id = '00000000-0000-4000-8000-000000002711' AND email = 'u49999@big.example'
  AND consumed_at IS NULL AND expires_at > now() RETURNING tenant_id, email;
ROLLBACK;
-- 3. A tenant's open invites (a likely group-sync read), large tenant.
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000002711', true);
EXPLAIN (ANALYZE, BUFFERS) SELECT email, expires_at FROM pending_invite
WHERE tenant_id = '00000000-0000-4000-8000-000000002711' AND consumed_at IS NULL AND expires_at > now();
ROLLBACK;
RESET ROLE;

-- 4. Re-invite upsert, as nava_onboard: a pending row and a consumed row.
SET ROLE nava_onboard;
BEGIN;
SELECT set_config('app.tenant_id', '00000000-0000-4000-8000-000000002711', true);
EXPLAIN (ANALYZE, BUFFERS) INSERT INTO pending_invite (tenant_id, email, expires_at)
VALUES ('00000000-0000-4000-8000-000000002711', 'u49999@big.example', now() + interval '30 days')
ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at
WHERE pending_invite.consumed_at IS NULL;
EXPLAIN (ANALYZE, BUFFERS) INSERT INTO pending_invite (tenant_id, email, expires_at)
VALUES ('00000000-0000-4000-8000-000000002711', 'u3@big.example', now() + interval '30 days')
ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at
WHERE pending_invite.consumed_at IS NULL;
ROLLBACK;
RESET ROLE;
