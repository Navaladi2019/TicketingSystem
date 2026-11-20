-- Back to scenario A: 60k of the 100k lack priority, at schema_version 1.
ALTER TABLE records DISABLE TRIGGER rfv_maint;
UPDATE records SET data = data - 'priority', schema_version = 1, version = 1
 WHERE tenant_id = u(1,1) AND substr(number,3)::int % 5 >= 2;
UPDATE records SET schema_version = 2, version = 1 WHERE tenant_id = u(1,1) AND substr(number,3)::int % 5 < 2;
ALTER TABLE records ENABLE TRIGGER rfv_maint;
TRUNCATE rfv_num, schema_backfill_failures, audit_log;
DROP TABLE IF EXISTS schema_backfill_pending;
INSERT INTO rfv_num SELECT tenant_id, u(1,99), id, 0, (data->>'priority')::float8 FROM records WHERE tenant_id = u(1,1) AND data ? 'priority';
