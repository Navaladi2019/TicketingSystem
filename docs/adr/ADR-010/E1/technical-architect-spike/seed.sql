SET session_replication_role = replica;
INSERT INTO tenants SELECT * FROM tenants WHERE false; -- shape check only
\set T '''0a000000-0000-4000-8000-000000000001'''
\set D '''d0000000-0000-4000-8000-000000000001'''
INSERT INTO entity_definitions (tenant_id,id,key,name,number_prefix,current_version) VALUES (:T,:D,'golden','golden','GLD',1);
CREATE TEMP TABLE grp(i int, id uuid);
INSERT INTO grp VALUES (0,'11111111-1111-4111-8111-111111111111'),(1,'22222222-2222-4222-8222-222222222222'),(2,'33333333-3333-4333-8333-333333333333'),(3,'55555555-5555-4555-8555-555555555555'),(4,'66666666-6666-4666-8666-666666666666');
INSERT INTO grp SELECT g, md5('g'||g)::uuid FROM generate_series(5,19) g;
INSERT INTO groups (tenant_id,id,keycloak_group_id,path) SELECT :T, id, 'kc'||i, '/t/'||:T||'/g'||i FROM grp;
CREATE OR REPLACE FUNCTION pick_user(n int, m int) RETURNS uuid LANGUAGE sql IMMUTABLE AS $$
  SELECT CASE n % m WHEN 0 THEN '0a000000-0000-4000-8000-0000000000c1'::uuid WHEN 1 THEN '44444444-4444-4444-8444-444444444444'::uuid ELSE md5('u'||(n % m))::uuid END $$;
INSERT INTO records (tenant_id,id,version,entity_definition_id,schema_version,number,state,requester_id,assignee_id,assignment_group_id,created_by,created_at,updated_at,data)
SELECT :T, md5('r'||i)::uuid, 1, :D, 1, 'GLD-'||i,
  (ARRAY['new','open','open','open','closed','closed','closed','closed','resolved','pending'])[1 + (i*7) % 10],
  pick_user(i, 20000),
  CASE WHEN i % 5 = 0 THEN NULL ELSE pick_user(i/5 + 13, 500) END,
  (SELECT id FROM grp WHERE grp.i = (s.i*31) % 20),
  pick_user(i, 20000),
  now() - make_interval(secs => (i * 39.37)::int), now() - make_interval(secs => (i * 37.11)::int % 31536000),
  jsonb_build_object('n', i)
FROM generate_series(1, 800000) s(i);
INSERT INTO record_shares (tenant_id,record_id,principal_kind,user_id)
SELECT DISTINCT :T::uuid, md5('r'||(1 + (j*7919) % 800000))::uuid, 'user', pick_user(j, 500) FROM generate_series(1,40000) j ON CONFLICT DO NOTHING;
INSERT INTO record_shares (tenant_id,record_id,principal_kind,group_id)
SELECT :T, md5('r'||(1 + (j*104729) % 800000))::uuid, 'group', (SELECT id FROM grp WHERE grp.i = j % 20) FROM generate_series(1,20000) j ON CONFLICT DO NOTHING;
SET session_replication_role = origin;
VACUUM (ANALYZE) records; VACUUM (ANALYZE) record_shares; ANALYZE groups; ANALYZE entity_definitions;
SELECT 'records', count(*) FROM records UNION ALL SELECT 'shares', count(*) FROM record_shares
UNION ALL SELECT 'req=c1', count(*) FROM records WHERE requester_id='0a000000-0000-4000-8000-0000000000c1'
UNION ALL SELECT 'asg=c1', count(*) FROM records WHERE assignee_id='0a000000-0000-4000-8000-0000000000c1'
UNION ALL SELECT 'asg=4444', count(*) FROM records WHERE assignee_id='44444444-4444-4444-8444-444444444444'
UNION ALL SELECT 'grp=2222', count(*) FROM records WHERE assignment_group_id='22222222-2222-4222-8222-222222222222'
UNION ALL SELECT 'share c1', count(*) FROM record_shares WHERE principal_id='0a000000-0000-4000-8000-0000000000c1';
