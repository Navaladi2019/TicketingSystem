\set n random(1, 100000)
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true);
UPDATE records SET data = data || '{"site":"s1"}', version = version + 1, updated_at = now()
 WHERE tenant_id = '00000001-0000-7000-8000-000000000001' AND id = u(1000000 + :n, 3);
COMMIT;
