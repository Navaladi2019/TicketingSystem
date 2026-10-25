-- T16 benchmark seed, run as the bootstrap superuser (RLS does not apply)
-- after db-roles and every migration. 10 large tenants at the acceptance
-- volume (50k principals, 5k groups, 8 memberships per principal, 500 org
-- units each) and 190 small tenants (100 principals, 10 groups, 5 units), so
-- tenants 1..10 hold ~96% of the rows. Ids are md5-derived: runs repeat.
\timing on
CREATE FUNCTION pg_temp.u(text) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$ SELECT md5($1)::uuid $$;

INSERT INTO tenants (id, state)
SELECT pg_temp.u('t' || t), 'active' FROM generate_series(1, 200) t;

CREATE TEMP TABLE shape AS
SELECT t, CASE WHEN t <= 10 THEN 50000 ELSE 100 END AS np,
          CASE WHEN t <= 10 THEN 5000 ELSE 10 END AS ng,
          CASE WHEN t <= 10 THEN 500 ELSE 5 END AS nou
FROM generate_series(1, 200) t;

-- One in three principals is brokered (one federated link); 2% disabled.
INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
SELECT pg_temp.u('t' || t), pg_temp.u('p' || t || ':' || p), pg_temp.u('kcu' || t || ':' || p)::text,
       'user' || p || '@tenant' || t || '.test', p % 50 <> 0,
       CASE WHEN p % 3 = 0 THEN jsonb_build_array(jsonb_build_object('identityProvider', 'idp-t' || t,
                'userId', 'ext-' || p, 'userName', 'user' || p)) ELSE '[]'::jsonb END,
       now()
FROM shape, generate_series(1, np) p;

INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
SELECT pg_temp.u('t' || t), pg_temp.u('g' || t || ':' || g), pg_temp.u('kcg' || t || ':' || g)::text,
       '/t/' || pg_temp.u('t' || t)::text || '/team-' || g
FROM shape, generate_series(1, ng) g;

-- 8 distinct groups per principal; principal 1 of each tenant is in up to
-- 200 (the worst case the resolver reads).
INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT pg_temp.u('t' || t), pg_temp.u('g' || t || ':' || (1 + (p * 7 + k * (ng / 8)) % ng)), pg_temp.u('p' || t || ':' || p)
FROM shape, generate_series(2, np) p, generate_series(0, 7) k;
INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT pg_temp.u('t' || t), pg_temp.u('g' || t || ':' || g), pg_temp.u('p' || t || ':1')
FROM shape, generate_series(1, least(ng, 200)) g;

-- A tree: unit 1 is the root, unit n hangs under unit n/10.
INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
SELECT pg_temp.u('t' || t), pg_temp.u('ou' || t || ':' || o),
       CASE WHEN o = 1 THEN NULL ELSE pg_temp.u('ou' || t || ':' || greatest(1, o / 10)) END,
       pg_temp.u('kco' || t || ':' || o)::text, 'Unit ' || o
FROM shape, generate_series(1, nou) o;

VACUUM (ANALYZE) principals, groups, group_members, org_units, tenants;

SELECT relname, n_live_tup FROM pg_stat_user_tables
WHERE relname IN ('principals', 'groups', 'group_members', 'org_units') ORDER BY 1;
