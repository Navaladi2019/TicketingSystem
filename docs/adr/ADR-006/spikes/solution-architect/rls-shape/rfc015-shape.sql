\set ON_ERROR_STOP on
CREATE ROLE nava_app LOGIN NOBYPASSRLS;
CREATE TABLE records (tenant_id uuid, id uuid, entity_definition_id uuid, updated_at timestamptz, data jsonb, PRIMARY KEY (tenant_id,id));
CREATE TABLE record_field_values (tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL, field_id uuid NOT NULL, record_id uuid NOT NULL,
  value_text text, value_num float8, value_ts timestamptz, value_bool boolean,
  PRIMARY KEY (tenant_id, entity_definition_id, field_id, record_id),
  FOREIGN KEY (tenant_id, record_id) REFERENCES records(tenant_id,id));
CREATE INDEX rfv_text ON record_field_values (tenant_id, entity_definition_id, field_id, value_text, record_id) WHERE value_text IS NOT NULL;
CREATE INDEX rfv_num  ON record_field_values (tenant_id, entity_definition_id, field_id, value_num, record_id) WHERE value_num IS NOT NULL;
CREATE INDEX rfv_ts   ON record_field_values (tenant_id, entity_definition_id, field_id, value_ts, record_id) WHERE value_ts IS NOT NULL;
CREATE INDEX rfv_bool ON record_field_values (tenant_id, entity_definition_id, field_id, value_bool, record_id) WHERE value_bool IS NOT NULL;
ALTER TABLE records ENABLE ROW LEVEL SECURITY; ALTER TABLE records FORCE ROW LEVEL SECURITY;
ALTER TABLE record_field_values ENABLE ROW LEVEL SECURITY; ALTER TABLE record_field_values FORCE ROW LEVEL SECURITY;
CREATE POLICY p ON records TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
CREATE POLICY p ON record_field_values TO nava_app USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
GRANT SELECT ON records, record_field_values TO nava_app;
ALTER ROLE nava_app SET plan_cache_mode = force_custom_plan;
-- 400 tenants: one with 200k, 399 with 500 each; def d; fields f1 text(site, 50 values), f2 num(priority 1..5), f3 ts, f4 bool
INSERT INTO records SELECT CASE WHEN g<=200000 THEN '00000000-0000-0000-0000-000000000001'::uuid ELSE md5('t'||(g%399))::uuid END,
  md5('r'||g)::uuid, '00000000-0000-0000-0000-0000000000dd', now()-random()*interval '365 days',
  jsonb_build_object('site','s'||(g%50),'priority',1+(g%5),'due',now()+(g%1000)*interval '1 hour','vip',(g%20=0))
FROM generate_series(1,399500) g;
INSERT INTO record_field_values SELECT tenant_id, entity_definition_id, '00000000-0000-0000-0000-0000000000f1', id, data->>'site', NULL,NULL,NULL FROM records;
INSERT INTO record_field_values SELECT tenant_id, entity_definition_id, '00000000-0000-0000-0000-0000000000f2', id, NULL,(data->>'priority')::float8,NULL,NULL FROM records;
INSERT INTO record_field_values SELECT tenant_id, entity_definition_id, '00000000-0000-0000-0000-0000000000f3', id, NULL,NULL,(data->>'due')::timestamptz,NULL FROM records;
INSERT INTO record_field_values SELECT tenant_id, entity_definition_id, '00000000-0000-0000-0000-0000000000f4', id, NULL,NULL,NULL,(data->>'vip')::bool FROM records;
VACUUM ANALYZE;
SET ROLE nava_app;
SELECT set_config('app.tenant_id','00000000-0000-0000-0000-000000000001',false);
\echo === text = (site=s7, 2% of tenant)
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT count(*) FROM record_field_values v WHERE v.tenant_id='00000000-0000-0000-0000-000000000001' AND v.entity_definition_id='00000000-0000-0000-0000-0000000000dd' AND v.field_id='00000000-0000-0000-0000-0000000000f1' AND v.value_text='s7';
\echo === text IN
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT count(*) FROM record_field_values v WHERE v.tenant_id='00000000-0000-0000-0000-000000000001' AND v.entity_definition_id='00000000-0000-0000-0000-0000000000dd' AND v.field_id='00000000-0000-0000-0000-0000000000f1' AND v.value_text = ANY('{s7,s8}');
\echo === float8 < 
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT count(*) FROM record_field_values v WHERE v.tenant_id='00000000-0000-0000-0000-000000000001' AND v.entity_definition_id='00000000-0000-0000-0000-0000000000dd' AND v.field_id='00000000-0000-0000-0000-0000000000f2' AND v.value_num < 2;
\echo === timestamptz range
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT count(*) FROM record_field_values v WHERE v.tenant_id='00000000-0000-0000-0000-000000000001' AND v.entity_definition_id='00000000-0000-0000-0000-0000000000dd' AND v.field_id='00000000-0000-0000-0000-0000000000f3' AND v.value_ts < now()+interval '10 hours';
\echo === bool =
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT count(*) FROM record_field_values v WHERE v.tenant_id='00000000-0000-0000-0000-000000000001' AND v.entity_definition_id='00000000-0000-0000-0000-0000000000dd' AND v.field_id='00000000-0000-0000-0000-0000000000f4' AND v.value_bool = true;
\echo === page: records joined by EXISTS, site=s7, ordered by updated_at
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT r.id FROM records r WHERE r.tenant_id='00000000-0000-0000-0000-000000000001' AND r.entity_definition_id='00000000-0000-0000-0000-0000000000dd'
  AND EXISTS (SELECT 1 FROM record_field_values v WHERE v.tenant_id=r.tenant_id AND v.entity_definition_id=r.entity_definition_id AND v.field_id='00000000-0000-0000-0000-0000000000f1' AND v.record_id=r.id AND v.value_text='s7')
  ORDER BY r.updated_at DESC, r.id DESC LIMIT 50;
\echo === control: jsonb ->> on records
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS) SELECT count(*) FROM records r WHERE r.tenant_id='00000000-0000-0000-0000-000000000001' AND r.data->>'site'='s7';
\echo === pg_input_is_valid
SELECT pg_input_is_valid('12x','float8') AS bad, pg_input_is_valid('2026-01-01','timestamptz') AS ok;
