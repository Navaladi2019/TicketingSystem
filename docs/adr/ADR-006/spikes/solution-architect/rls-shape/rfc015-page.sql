SET ROLE nava_app;
SELECT set_config('app.tenant_id','00000000-0000-0000-0000-000000000001',false) \g /dev/null
\echo === page site=s7 (2%) warm
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS, SUMMARY ON, TIMING OFF) SELECT r.id FROM records r WHERE r.tenant_id='00000000-0000-0000-0000-000000000001' AND r.entity_definition_id='00000000-0000-0000-0000-0000000000dd'
  AND EXISTS (SELECT 1 FROM record_field_values v WHERE v.tenant_id=r.tenant_id AND v.entity_definition_id=r.entity_definition_id AND v.field_id='00000000-0000-0000-0000-0000000000f1' AND v.record_id=r.id AND v.value_text='s7')
  ORDER BY r.updated_at DESC, r.id DESC LIMIT 50;
\echo === page priority<3 (40%) warm
EXPLAIN (ANALYZE, COSTS OFF, BUFFERS, SUMMARY ON, TIMING OFF) SELECT r.id FROM records r WHERE r.tenant_id='00000000-0000-0000-0000-000000000001' AND r.entity_definition_id='00000000-0000-0000-0000-0000000000dd'
  AND EXISTS (SELECT 1 FROM record_field_values v WHERE v.tenant_id=r.tenant_id AND v.entity_definition_id=r.entity_definition_id AND v.field_id='00000000-0000-0000-0000-0000000000f2' AND v.record_id=r.id AND v.value_num < 3)
  ORDER BY r.updated_at DESC, r.id DESC LIMIT 50;
