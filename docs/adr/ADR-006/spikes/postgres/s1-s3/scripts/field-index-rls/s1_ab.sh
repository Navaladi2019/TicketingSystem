#!/bin/sh
# TICK-714 R10 and coerce A/B: alternating 30 s insert runs, 10 filterable
# fields, so table growth and host noise hit both arms equally.
set -u
cd /bench/fi
su() { psql -X -q -At -U pgadmin -d nava -c "$1"; }
fk() { for t in rfv_text rfv_num rfv_ts rfv_bool record_search_terms; do
    if [ "$1" = off ]; then su "ALTER TABLE $t DROP CONSTRAINT IF EXISTS ${t}_field_fk"
    else su "ALTER TABLE $t ADD CONSTRAINT ${t}_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id)"; fi; done; }
su "UPDATE record_index_fields SET state = CASE WHEN kind = 'filter' THEN 'ready' ELSE 'removing' END WHERE tenant_id = md5('tenant0')::uuid AND entity_definition_id = md5('ed0/0')::uuid"
for i in 1 2 3; do
    fk on;  su CHECKPOINT; sh fq.sh write insert "10f,fk,round$i" 30 8
    fk off; su CHECKPOINT; sh fq.sh write insert "10f,no-fk,round$i" 30 8
done
fk on
su "CHECKPOINT"; sh fq.sh write insert "10f,fk,coerce-SET(D19 as written)" 30 8
psql -X -q -U pgadmin -d nava -f /bench/fi/24_coerce_inline.sql
for i in 1 2; do
    su "CHECKPOINT"; sh fq.sh write insert "10f,fk,coerce-inlined,round$i" 30 8
    su "SELECT 1"; psql -X -q -U pgadmin -d nava -c "DO \$\$ DECLARE s text; BEGIN SELECT pg_get_functiondef('record_field_values_sync'::regproc) INTO s; EXECUTE replace(s, 'record_field_coerce_inl(', 'record_field_coerce('); END \$\$"
    su "CHECKPOINT"; sh fq.sh write insert "10f,fk,coerce-SET,round$i" 30 8
    psql -X -q -U pgadmin -d nava -f /bench/fi/24_coerce_inline.sql
done
su "ALTER TABLE records DISABLE TRIGGER record_field_values_sync"; su CHECKPOINT
sh fq.sh write insert "no-trigger,end" 30 8
su "ALTER TABLE records ENABLE TRIGGER record_field_values_sync"
echo AB-DONE
