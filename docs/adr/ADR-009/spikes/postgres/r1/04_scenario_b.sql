-- Scenario B: only 20 records of the 100k type lack priority (initially, or
-- what is left in failed_records). Default 3 is stored nowhere.
ALTER TABLE records DISABLE TRIGGER rfv_maint;
UPDATE records SET data = data || jsonb_build_object('priority', (ARRAY[1,2,4,5])[1 + (substr(number,3)::int % 4)]), schema_version = 2
WHERE tenant_id = u(1,1) AND NOT data ? 'priority' AND substr(number,3)::int % 3000 <> 7;
ALTER TABLE records ENABLE TRIGGER rfv_maint;
TRUNCATE rfv_num;
INSERT INTO rfv_num SELECT tenant_id, u(1,99), id, 0, (data->>'priority')::float8 FROM records WHERE tenant_id = u(1,1) AND data ? 'priority';
INSERT INTO schema_backfill_failures SELECT tenant_id, entity_definition_id, 3, id, 'validation' FROM records WHERE tenant_id = u(1,1) AND NOT data ? 'priority';
