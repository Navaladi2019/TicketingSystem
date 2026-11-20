-- SP-2 (TICK-933) item 3: above how many total ReadBranches branches does the
-- index-per-branch list (UNION ALL, internal/records/store/list.go
-- branchedPage) stop beating the OR form (CompileRead, PageStatement)?
-- Run by branches.sh on bench/records-list's 800k seed (P3-01) with the
-- TICK-640 list indexes, as nava_app under forced RLS, JIT off.
--
-- The branch text is ReadBranches' (internal/authz/acl.go): each branch is
-- (d_i AND NOT d_1 ... AND NOT d_{i-1}), every term two-valued
-- ("col IS NOT NULL AND col = id"). Two families of N branches for one
-- caller of the large tenant (400,000 rows, 80,000 per entity type):
--   sparse: requester = me, assignee = me, then assignee = a named user
--           (one per branch): about 16 rows each, the "team member" policy.
--           The OR form must walk far down updated_at to fill a page.
--   heavy:  as sparse, but the caller is the heaviest requester of the
--           tenant (bench_heavy n = 1, thousands of rows): the case that
--           cost the OR form 38-166 ms in docs/data/records-list-indexes.md.
--   dense:  requester = me, assignee = me, then assignment_group = a named
--           group (then group AND state when the 50 groups run out):
--           about 1,600 rows each. The OR form fills a page early.
-- Page 1, sort -updated_at, LIMIT 51 (limit 50 + 1). The joins to
-- entity_definitions and groups on the <= 51 result rows are left out: they
-- are the same for both forms.
\pset pager off
SET jit = off;

CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC);
CREATE INDEX records_list_requester ON records (tenant_id, entity_definition_id, requester_id DESC, updated_at DESC, id DESC);
CREATE INDEX records_list_assignee ON records (tenant_id, entity_definition_id, assignee_id DESC, updated_at DESC, id DESC);
CREATE INDEX records_list_group ON records (tenant_id, entity_definition_id, assignment_group_id DESC, updated_at DESC, id DESC);
CREATE INDEX records_list_group_state ON records (tenant_id, entity_definition_id, assignment_group_id DESC, state DESC, updated_at DESC, id DESC);
ANALYZE records;
SELECT c.relname, pg_prewarm(c.oid) AS blocks FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relname LIKE 'record%' AND c.relkind IN ('r', 'i') ORDER BY c.relname;

-- The N disjuncts of a family, as ReadBranches renders them.
CREATE FUNCTION bench_disjuncts(family text, n int, me uuid) RETURNS text[] LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE d text[] := '{}'; id uuid; st text;
BEGIN
    d := d || format('(r.requester_id IS NOT NULL AND r.requester_id = %L::uuid)', me);
    IF n >= 2 THEN d := d || format('(r.assignee_id IS NOT NULL AND r.assignee_id = %L::uuid)', me); END IF;
    FOR k IN 3 .. n LOOP
        IF family IN ('sparse', 'heavy') THEN
            id := md5('u0/' || ((k * 37) % 5000))::uuid;
            d := d || format('(r.assignee_id IS NOT NULL AND r.assignee_id = %L::uuid)', id);
        ELSE
            id := md5('g0/' || (k % 50))::uuid;
            IF k < 53 THEN
                d := d || format('(r.assignment_group_id IS NOT NULL AND r.assignment_group_id = %L::uuid)', id);
            ELSE
                st := (ARRAY['new', 'in_progress', 'on_hold', 'closed'])[1 + ((k / 50) % 4)];
                d := d || format('(r.assignment_group_id IS NOT NULL AND r.assignment_group_id = %L::uuid AND (r.state = ANY(%L::text[])))', id, ARRAY[st]);
            END IF;
        END IF;
    END LOOP;
    RETURN d;
END $$;

CREATE FUNCTION bench_sql(form text, d text[], tenant uuid, def uuid) RETURNS text LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE
    cols CONSTANT text := 'r.tenant_id, r.id, r.entity_definition_id, r.number, r.state, r.requester_id, r.assignee_id, r.assignment_group_id, r.created_at, r.updated_at, r.data';
    head text := format('r.tenant_id = %L::uuid AND r.entity_definition_id = %L::uuid', tenant, def);
    arms text[] := '{}'; b text;
BEGIN
    IF form = 'or' THEN
        RETURN 'SELECT ' || cols || ' FROM records r WHERE ' || head || ' AND (' || array_to_string(d, ' OR ')
            || ') ORDER BY r.updated_at DESC, r.id DESC LIMIT 51';
    END IF;
    FOR i IN 1 .. cardinality(d) LOOP
        b := d[i];
        FOR j IN 1 .. i - 1 LOOP b := b || ' AND NOT ' || d[j]; END LOOP;
        arms := arms || ('(SELECT ' || cols || ' FROM records r WHERE ' || head || ' AND (' || b
            || ') ORDER BY r.updated_at DESC, r.id DESC LIMIT 51)');
    END LOOP;
    RETURN 'SELECT * FROM (' || array_to_string(arms, E'\nUNION ALL\n') || E'\nORDER BY updated_at DESC, id DESC LIMIT 51) x';
END $$;

CREATE TABLE bench_result (family text, n int, form text, rep int, plan_ms float8, exec_ms float8, rows int, shared_hit bigint, sql_bytes int);
GRANT INSERT, SELECT ON bench_result TO nava_app;

-- EXPLAIN (ANALYZE, BUFFERS, TIMING OFF): planning and execution as the
-- server reports them. The server runs in QueryExecModeExec, so every list
-- request plans its statement afresh: planning is part of the cost.
CREATE FUNCTION bench_run(family text, ns int[], reps int) RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    me uuid; tenant uuid := md5('tenant0')::uuid; def uuid := md5('ed0/0')::uuid;
    d text[]; q text; j jsonb; k int; f text;
BEGIN
    IF family = 'heavy' THEN
        SELECT user_id INTO me FROM bench_heavy WHERE bench_heavy.n = 1;
    ELSE
        SELECT user_id INTO me FROM bench_actor WHERE kind = 'large' AND bench_actor.n = 1;
    END IF;
    PERFORM set_config('app.tenant_id', tenant::text, false);
    FOREACH k IN ARRAY ns LOOP
        d := bench_disjuncts(family, k, me);
        FOREACH f IN ARRAY CASE WHEN k > 128 THEN ARRAY['or'] ELSE ARRAY['branches', 'or'] END LOOP
            q := bench_sql(f, d, tenant, def);
            FOR rep IN 0 .. reps LOOP  -- rep 0 warms up and is dropped
                EXECUTE 'EXPLAIN (ANALYZE, BUFFERS, TIMING OFF, FORMAT JSON) ' || q INTO j;
                IF rep > 0 THEN
                    INSERT INTO bench_result VALUES (family, k, f, rep, (j -> 0 ->> 'Planning Time')::float8,
                        (j -> 0 ->> 'Execution Time')::float8, (j -> 0 -> 'Plan' ->> 'Actual Rows')::int,
                        (j -> 0 -> 'Plan' ->> 'Shared Hit Blocks')::bigint, octet_length(q));
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;
END $$;
-- bench_actor and bench_heavy are superuser tables of the seed.
GRANT SELECT ON bench_actor, bench_heavy TO nava_app;

\echo '== rows the caller can see in the entity type (superuser count)'
SELECT f AS family, k AS n, (SELECT count(*) FROM records r WHERE r.tenant_id = md5('tenant0')::uuid AND r.entity_definition_id = md5('ed0/0')::uuid
    AND (r.requester_id = a.user_id OR r.assignee_id = a.user_id
         OR (f = 'sparse' AND r.assignee_id IN (SELECT md5('u0/' || ((i * 37) % 5000))::uuid FROM generate_series(3, k) i))
         OR (f = 'dense' AND r.assignment_group_id IN (SELECT md5('g0/' || (i % 50))::uuid FROM generate_series(3, least(k, 52)) i)))) visible
FROM bench_actor a, unnest(ARRAY['sparse', 'dense']) f, unnest(ARRAY[2, 4, 8, 16, 32, 64]) k WHERE a.kind = 'large' AND a.n = 1 ORDER BY 1, 2;

SET ROLE nava_app;
SELECT bench_run('sparse', ARRAY[2, 4, 6, 8, 12, 16, 24, 32, 48, 64, 96, 128], 5);
SELECT bench_run('dense', ARRAY[2, 4, 6, 8, 12, 16, 24, 32, 48, 64, 96, 128], 5);
SELECT bench_run('heavy', ARRAY[2, 4, 6, 8, 12, 16, 24, 32, 48, 64, 96, 128], 5);
-- The CompileRead fallback alone at the per-document leaf cap: one
-- disjunct per leaf is the most a document can render to (branched text
-- would be 100 MB at 1,024).
SELECT bench_run(f, ARRAY[256, 512, 1024], 5) FROM unnest(ARRAY['sparse', 'heavy']) f;
RESET ROLE;

\echo '== median of 5 runs, ms: planning + execution = total'
SELECT family, n, form, round(percentile_cont(0.5) WITHIN GROUP (ORDER BY plan_ms)::numeric, 2) plan,
       round(percentile_cont(0.5) WITHIN GROUP (ORDER BY exec_ms)::numeric, 2) exec,
       round(percentile_cont(0.5) WITHIN GROUP (ORDER BY plan_ms + exec_ms)::numeric, 2) total,
       round(max(plan_ms + exec_ms)::numeric, 2) max_total, max(rows) rows, max(shared_hit) hit_blocks, max(sql_bytes) sql_bytes
FROM bench_result GROUP BY 1, 2, 3 ORDER BY 1, 2, 3;

\echo '== branched minus OR, median total ms (negative = branches win)'
SELECT family, n,
       round((percentile_cont(0.5) WITHIN GROUP (ORDER BY plan_ms + exec_ms) FILTER (WHERE form = 'branches')
            - percentile_cont(0.5) WITHIN GROUP (ORDER BY plan_ms + exec_ms) FILTER (WHERE form = 'or'))::numeric, 2) branches_minus_or
FROM bench_result GROUP BY 1, 2 ORDER BY 1, 2;

\echo '== heavy requester: rows in the entity type'
SELECT h.nrows AS all_types, (SELECT count(*) FROM records r WHERE r.tenant_id = h.tenant_id AND r.entity_definition_id = md5('ed0/0')::uuid AND r.requester_id = h.user_id) AS this_type
FROM bench_heavy h WHERE h.n = 1;
\echo '== plans: heavy n=8, OR form (TIMING on)'
SET ROLE nava_app;
SELECT set_config('app.tenant_id', md5('tenant0')::uuid::text, false);
SELECT bench_sql('or', bench_disjuncts('heavy', 8, (SELECT user_id FROM bench_heavy WHERE n = 1)), md5('tenant0')::uuid, md5('ed0/0')::uuid) AS q \gset
EXPLAIN (ANALYZE, BUFFERS) :q;
RESET ROLE;
\echo '== plans: sparse n=8, both forms (TIMING on)'
SET ROLE nava_app;
SELECT set_config('app.tenant_id', md5('tenant0')::uuid::text, false);
SELECT bench_sql('branches', bench_disjuncts('sparse', 8, (SELECT user_id FROM bench_actor WHERE kind = 'large' AND n = 1)), md5('tenant0')::uuid, md5('ed0/0')::uuid) AS q \gset
EXPLAIN (ANALYZE, BUFFERS) :q;
SELECT bench_sql('or', bench_disjuncts('sparse', 8, (SELECT user_id FROM bench_actor WHERE kind = 'large' AND n = 1)), md5('tenant0')::uuid, md5('ed0/0')::uuid) AS q \gset
EXPLAIN (ANALYZE, BUFFERS) :q;
RESET ROLE;
