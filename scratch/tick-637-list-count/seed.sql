-- TICK-637 synthetic data, run as the superuser on a migrated `nava` database.
-- Tenant L: 1,000,000 incident + 100,000 request records, 5,000 users, 100 groups.
-- Tenant M:   100,000 incident +  10,000 request records, 1,000 users,  50 groups.
-- 20 small tenants: 1,000 incident records each, 50 users, 10 groups.
-- Every user is in 3 groups. 15% of records are open, independent of time.
-- created_at = 2026-11-01 - 730 d * random()^3; updated_at >= created_at.
-- Shares: 2% of records shared with one user; 10% shared with one group,
-- half of those (5% of all records) with the tenant's group 0 ("partners").
SELECT setseed(0.637);
SET max_parallel_workers_per_gather = 0;
-- Seed only: skip the audit commit-check triggers (and FK triggers; the data is built consistent).
SET session_replication_role = replica;

CREATE TEMP TABLE t_spec (tk text, tid uuid, users int, ngroups int, inc int, req int);
INSERT INTO t_spec VALUES
  ('L', '00000000-0000-4000-8000-00000000000a', 5000, 100, 1000000, 100000),
  ('M', '00000000-0000-4000-8000-00000000000b', 1000,  50,  100000,  10000);
INSERT INTO t_spec SELECT 's'||i, ('00000000-0000-4000-8000-0000000001'||lpad(i::text,2,'0'))::uuid, 50, 10, 1000, 0
  FROM generate_series(1,20) i;

INSERT INTO tenants (id, state) SELECT tid, 'active' FROM t_spec;

-- Deterministic ids: user u of tenant -> md5-derived uuid.
CREATE FUNCTION pg_temp.uid(t uuid, kind text, n int) RETURNS uuid
  LANGUAGE sql IMMUTABLE AS $$ SELECT md5(t::text || kind || n)::uuid $$;

INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, synced_at)
SELECT tid, pg_temp.uid(tid,'u',u), 'kc-'||tk||'-'||u, 'u'||u||'@'||tk||'.test', true, now()
  FROM t_spec, generate_series(0, users-1) u;

INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
SELECT tid, pg_temp.uid(tid,'g',g), 'kcg-'||tk||'-'||g, '/t/'||tid||'/g'||g
  FROM t_spec, generate_series(0, ngroups-1) g;

INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT DISTINCT tid, pg_temp.uid(tid,'g',(u*7 + k*13) % ngroups), pg_temp.uid(tid,'u',u)
  FROM t_spec, generate_series(0, users-1) u, generate_series(0,2) k;

INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
SELECT tid, pg_temp.uid(tid,'d',0), 'incident', 'Incident', 'INC', 1 FROM t_spec
UNION ALL
SELECT tid, pg_temp.uid(tid,'d',1), 'request', 'Request', 'REQ', 1 FROM t_spec;

-- Records. Built per tenant and type into a temp table first, then copied in
-- created_at order would bias physical order; insert in random order instead.
CREATE TEMP TABLE t_rec AS
SELECT s.tid, d.n AS dn, i, s.users, s.ngroups,
       timestamptz '2026-11-01' - interval '730 days' * power(random(),3) AS created_at,
       random() AS r_state, random() AS r_req, random() AS r_asg, random() AS r_asgu,
       random() AS r_grp, random() AS r_grpn, random() AS r_upd
  FROM t_spec s
  CROSS JOIN LATERAL (VALUES (0, s.inc), (1, s.req)) d(n, cnt)
  CROSS JOIN LATERAL generate_series(1, d.cnt) i;

INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
                     requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data)
SELECT tid, gen_random_uuid(), 1, pg_temp.uid(tid,'d',dn), 1,
       CASE dn WHEN 0 THEN 'INC' ELSE 'REQ' END || lpad(i::text, 7, '0'),
       CASE WHEN r_state < 0.10 THEN 'new' WHEN r_state < 0.15 THEN 'in_progress' ELSE 'closed' END,
       pg_temp.uid(tid,'u', floor(power(r_req,2) * users)::int),
       CASE WHEN r_asg < 0.9 THEN pg_temp.uid(tid,'u', floor(r_asgu * users)::int) END,
       CASE WHEN r_grp < 0.95 THEN pg_temp.uid(tid,'g', floor(r_grpn * ngroups)::int) END,
       pg_temp.uid(tid,'u', floor(power(r_req,2) * users)::int),
       created_at,
       least(created_at + interval '30 days' * r_upd, timestamptz '2026-11-01'),
       jsonb_build_object(
         'short_description', 'Synthetic record ' || i || ' ' || md5(i::text),
         'description', repeat(md5(i::text || 'x'), 24),
         'priority', (1 + floor(r_upd * 4))::int::text,
         'category', 'category-' || (i % 20),
         'impact', (i % 3) + 1)
  FROM t_rec ORDER BY random();
DROP TABLE t_rec;

-- Shares.
INSERT INTO record_shares (tenant_id, record_id, principal_kind, user_id, granted_at)
SELECT r.tenant_id, r.id, 'user', pg_temp.uid(r.tenant_id,'u', floor(random() * s.users)::int), now()
  FROM records r JOIN t_spec s ON s.tid = r.tenant_id
 WHERE random() < 0.02;

INSERT INTO record_shares (tenant_id, record_id, principal_kind, group_id, granted_at)
SELECT tenant_id, id, 'group',
       pg_temp.uid(tenant_id,'g', CASE WHEN rg < 0.5 THEN 0 ELSE 1 + floor(random() * (ng - 1))::int END), now()
  FROM (SELECT r.tenant_id, r.id, random() AS rs, random() AS rg, s.ngroups AS ng
          FROM records r JOIN t_spec s ON s.tid = r.tenant_id) x
 WHERE rs < 0.10;

VACUUM (ANALYZE) tenants, principals, groups, group_members, entity_definitions, records, record_shares;
