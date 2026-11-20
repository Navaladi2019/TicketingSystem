\set n random(1, 100000)
BEGIN;
SELECT set_config('app.tenant_id', '00000001-0000-7000-8000-000000000001', true);
INSERT INTO record_shares VALUES ('00000001-0000-7000-8000-000000000001', u(1000000 + :n, 3), gen_random_uuid());
COMMIT;
