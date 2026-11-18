-- SP-1b (TICK-969) item 5: server-side cost of the single-record check
-- (parse, plan, execute; dynamic SQL so every call is planned, like
-- force_custom_plan). Run as nava_app: RLS applies. 2,000 calls per case.
-- Needs (as superuser): CREATE TABLE bench_ou_ids AS SELECT pre, id FROM bench_ou;
-- GRANT SELECT ON bench_ou_ids, perf_tenant TO nava_app;
\set ON_ERROR_STOP on
BEGIN;
SELECT set_config('app.tenant_id', (SELECT tenant_id::text FROM perf_tenant WHERE tn = 0), true) \gset
DO $$
DECLARE
  t uuid := current_setting('app.tenant_id')::uuid;
  c record;
  d float8[];
  t0 timestamptz;
  ok bool;
  hits int;
  a uuid;
  r uuid;
BEGIN
  FOR c IN
    SELECT * FROM (VALUES
      ('area, 10 units, inside', 3, 12), ('area, 10 units, outside', 3, 1017),
      ('district, 101 units, inside', 2, 102), ('district, 101 units, outside', 2, 1017),
      ('region, 1011 units, inside', 1, 1011), ('region, 1011 units, outside', 1, 1017),
      ('root, 10111 units, inside', 0, 10110)) v(name, anc_pre, rec_pre)
  LOOP
    d := '{}'; hits := 0;
    SELECT id INTO a FROM bench_ou_ids WHERE pre = c.anc_pre;
    SELECT id INTO r FROM bench_ou_ids WHERE pre = c.rec_pre;
    FOR i IN 1..2000 LOOP
      t0 := clock_timestamp();
      EXECUTE 'SELECT EXISTS (SELECT 1 FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = ANY($2::uuid[]) AND descendant_id = $3)'
        INTO ok
        USING t, ARRAY[a], r;
      d := d || extract(epoch FROM clock_timestamp() - t0) * 1000;
      IF ok THEN hits := hits + 1; END IF;
    END LOOP;
    RAISE NOTICE '% | hits % | p50 % | p95 % | p99 % ms', c.name, hits,
      round((SELECT percentile_disc(0.5) WITHIN GROUP (ORDER BY x) FROM unnest(d) x)::numeric, 3),
      round((SELECT percentile_disc(0.95) WITHIN GROUP (ORDER BY x) FROM unnest(d) x)::numeric, 3),
      round((SELECT percentile_disc(0.99) WITHIN GROUP (ORDER BY x) FROM unnest(d) x)::numeric, 3);
  END LOOP;
END $$;
ROLLBACK;
