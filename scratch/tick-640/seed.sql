-- TICK-640 seed, read by setup.go -seed (one statement per ";" line end).
-- P3-01's 800k shape on the real schema, as P3-36's TestRecordsACLPerf800k:
-- tenant 0 has 400,000 records and 5,000 users; tenants 1..399 have 1,003
-- (tn <= 202) or 1,002 records and 20 users; 50 groups per tenant plus an
-- agents and an admins group; 3 groups per user; 2% of records shared.
-- The superuser bypasses RLS; session_replication_role = replica skips the
-- audit commit check and the foreign keys for the bulk load.

SET max_parallel_workers_per_gather = 0;

SET session_replication_role = replica;

CREATE TABLE perf_tenant (tn int PRIMARY KEY, tenant_id uuid NOT NULL, nrec int NOT NULL, nusers int NOT NULL,
    agents uuid NOT NULL, admins uuid NOT NULL, def uuid NOT NULL);

INSERT INTO perf_tenant
 SELECT tn, md5('t' || tn)::uuid, CASE WHEN tn = 0 THEN 400000 WHEN tn <= 202 THEN 1003 ELSE 1002 END,
        CASE WHEN tn = 0 THEN 5000 ELSE 20 END, md5('agents/' || tn)::uuid, md5('admins/' || tn)::uuid, md5('def/' || tn)::uuid
 FROM generate_series(0, 399) tn;

INSERT INTO tenants (id, state) SELECT tenant_id, 'active' FROM perf_tenant;

INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
 SELECT tenant_id, def, 'ticket', 'Ticket', 'TKT', 1 FROM perf_tenant;

INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
 SELECT tenant_id, agents, 'perf-agents-' || tn, '/t/' || tenant_id || '/agents' FROM perf_tenant
 UNION ALL SELECT tenant_id, admins, 'perf-admins-' || tn, '/t/' || tenant_id || '/admins' FROM perf_tenant;

SELECT setseed(0.339);

INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
 SELECT t.tenant_id, md5('u' || t.tn || '/' || u)::uuid, 'perf-u' || t.tn || '-' || u, 'u' || u || '@t' || t.tn || '.test', true, '[]', now()
 FROM perf_tenant t, generate_series(0, t.nusers - 1) u;

INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
 SELECT t.tenant_id, md5('g' || t.tn || '/' || g)::uuid, 'perf-g' || t.tn || '-' || g, '/t/' || t.tenant_id || '/g' || g
 FROM perf_tenant t, generate_series(0, 49) g;

INSERT INTO group_members (tenant_id, group_id, principal_id)
 SELECT t.tenant_id, md5('g' || t.tn || '/' || ((u + k) % 50))::uuid, md5('u' || t.tn || '/' || u)::uuid
 FROM perf_tenant t, generate_series(0, t.nusers - 1) u, unnest(ARRAY[0, 17, 34]) k;

INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
 SELECT x.tenant_id,
        md5('r' || x.tn || '/' || x.n)::uuid,
        1, x.def, 1, 'TKT' || lpad(x.n::text, 7, '0'),
        CASE WHEN x.r1 < 0.15 THEN (ARRAY['new', 'in_progress', 'on_hold'])[1 + floor(x.r2 * 3)::int] ELSE 'closed' END,
        md5('u' || x.tn || '/' || floor(x.nusers * x.r3 * x.r3)::int)::uuid,
        CASE WHEN x.r4 < 0.9 THEN md5('u' || x.tn || '/' || floor(x.nusers * x.r5)::int)::uuid END,
        md5('g' || x.tn || '/' || floor(50 * x.r6)::int)::uuid,
        md5('u' || x.tn || '/' || floor(x.nusers * x.r3 * x.r3)::int)::uuid,
        x.upd - x.r7 * interval '30 days',
        x.upd,
        jsonb_build_object(
            'priority', CASE WHEN x.r8 < 0.05 THEN '1' WHEN x.r8 < 0.20 THEN '2' WHEN x.r8 < 0.70 THEN '3'
                             WHEN x.r8 < 0.90 THEN '4' ELSE '5' END,
            'category', 'category-' || floor(x.r9 * 20)::int,
            'impact', 1 + floor(x.r2 * 3)::int,
            'short_description', 'Record ' || x.n || ' ' || md5(x.n::text),
            'description', repeat(md5(x.tn || ':' || x.n), 27),
            'tags', jsonb_build_array('t' || floor(x.r5 * 50)::int, 't' || floor(x.r6 * 50)::int)),
        0
 FROM (
     SELECT t.tn, t.tenant_id, t.nusers, t.def, n,
            random() AS r1, random() AS r2, random() AS r3, random() AS r4, random() AS r5,
            random() AS r6, random() AS r7, random() AS r8, random() AS r9,
            timestamptz '2026-10-01 00:00:00+00' - power(random(), 3) * interval '365 days' AS upd
     FROM perf_tenant t, generate_series(1, t.nrec) n
     ORDER BY t.tn, n
 ) x;

INSERT INTO record_counters (tenant_id, entity_definition_id, next) SELECT tenant_id, def, nrec + 1 FROM perf_tenant;

INSERT INTO record_shares (tenant_id, record_id, principal_kind, user_id)
 SELECT r.tenant_id, r.id, 'user', md5('u' || t.tn || '/' || floor(t.nusers * random())::int)::uuid
 FROM records r JOIN perf_tenant t USING (tenant_id)
 WHERE random() < 0.02
 ON CONFLICT DO NOTHING;

SET session_replication_role = origin;

VACUUM (ANALYZE) records;

VACUUM (ANALYZE) record_shares;

VACUUM (ANALYZE) principals;

VACUUM (ANALYZE) groups;

VACUUM (ANALYZE) group_members;

VACUUM (ANALYZE) entity_definitions;

CREATE EXTENSION IF NOT EXISTS pg_prewarm;

