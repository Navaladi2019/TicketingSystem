-- Stored priority for the 40% that hold it: {1,2,4,5}, never the new default 3.
ALTER TABLE records DISABLE TRIGGER rfv_maint;
UPDATE records SET data = data || jsonb_build_object('priority', (ARRAY[1,2,4,5])[1 + (number_i % 4)])
FROM (SELECT id rid, substr(number,3)::int number_i FROM records WHERE tenant_id = u(1,1) AND data ? 'priority') s
WHERE tenant_id = u(1,1) AND id = s.rid;
ALTER TABLE records ENABLE TRIGGER rfv_maint;
TRUNCATE rfv_num;
INSERT INTO rfv_num SELECT tenant_id, u(1,99), id, 0, (data->>'priority')::float8 FROM records WHERE tenant_id = u(1,1) AND data ? 'priority';
