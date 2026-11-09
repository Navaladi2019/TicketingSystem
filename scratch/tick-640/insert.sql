-- TICK-640 write amplification: one record INSERT shaped like a create in the
-- 400k tenant (about 1 KB of data). Run as the superuser with
-- session_replication_role = replica (PGOPTIONS) so the deferred audit
-- commit check and the foreign keys, which the index set does not change,
-- stay out: the latency difference between two runs is index maintenance.
\set n random(1, 2000000000)
\set u random(0, 4999)
\set g random(0, 49)
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
VALUES ('809d4580-aaed-4156-5abc-38d58f77f840', gen_random_uuid(), 1, '22a88e5c-3a76-b13d-e14d-5bdef46b3dc2', 1,
    'TKB' || gen_random_uuid(), 'new', md5('u0/' || :u)::uuid, md5('u0/' || ((:u + 7) % 5000))::uuid, md5('g0/' || :g)::uuid,
    md5('u0/' || :u)::uuid, now(), now(),
    jsonb_build_object('priority', '3', 'category', 'category-1', 'impact', 2,
        'short_description', 'Bench ' || :n, 'description', repeat(md5(:n::text), 27), 'tags', jsonb_build_array('t1', 't2')),
    0);
