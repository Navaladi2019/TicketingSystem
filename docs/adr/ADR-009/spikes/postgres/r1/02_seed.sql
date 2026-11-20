-- 10 tenants x 100k = 1.0M records. Tenant 1, type T = the 100k type under test.
-- priority: 40% hold it explicitly (1..5), 60% lack it (written before v3 added default 3).
CREATE OR REPLACE FUNCTION u(i bigint, salt int) RETURNS uuid LANGUAGE sql IMMUTABLE AS
$$ SELECT (lpad(to_hex(salt),8,'0') || '-0000-7000-8000-' || lpad(to_hex(i),12,'0'))::uuid $$;
INSERT INTO fields VALUES (u(1,1), u(1,99), u(1,2), 'priority');
ALTER TABLE records DISABLE TRIGGER rfv_maint;
INSERT INTO records
SELECT u(t,1), u(t*1000000+i, 3), u(t,2), 1, CASE WHEN i % 5 < 2 THEN 2 ELSE 1 END, 'T-'||i,
  (ARRAY['new','in_progress','on_hold','closed'])[1 + i % 4], u(i % 500, 4), u(i % 50, 5), u(i % 20, 6),
  now() - (i || ' minutes')::interval, now() - ((i * 7919) % 100000 || ' minutes')::interval,
  jsonb_build_object('title', 'Ticket '||i, 'description', repeat(md5(i::text), 12), 'site', 's'||(i%40),
    'category', 'c'||(i%12), 'impact', i%3, 'tags', jsonb_build_array('a'||(i%7),'b'||(i%11)),
    'due', '2026-12-01', 'reporter_note', repeat('x', 80))
  || CASE WHEN i % 5 < 2 THEN jsonb_build_object('priority', 1 + i % 5 ) ELSE '{}'::jsonb END,
  NULL, 0
FROM generate_series(1,10) t, generate_series(1,100000) i;
ALTER TABLE records ENABLE TRIGGER rfv_maint;
INSERT INTO rfv_num SELECT tenant_id, u(1,99), id, 0, (data->>'priority')::float8
  FROM records WHERE tenant_id = u(1,1) AND data ? 'priority';
VACUUM ANALYZE;
