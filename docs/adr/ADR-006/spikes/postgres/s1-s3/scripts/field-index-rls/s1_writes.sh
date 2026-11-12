#!/bin/sh
# TICK-714 S1 writes: insert and update tps, 8 clients, 60 s, nava_app via
# PgBouncer, into the 400k type, for 0 / 5 / 10 filterable fields, with and
# without the D3 registry FK (R10), plus 10 filterable + 3 searchable (S2).
# The registry rows of the type are switched with state (ready = maintained,
# removing = ignored by the trigger), so no DDL between runs except the FK.
set -u
cd /bench/fi
SECS=${SECS:-60}
su() { psql -X -q -At -U pgadmin -d nava -c "$1"; }
T="md5('tenant0')::uuid"; E="md5('ed0/0')::uuid"
fields() { # fields <filter keys...>: those are ready, every other field of the type is removing
    keys=$(printf "'%s'," "$@"); keys=${keys%,}
    su "UPDATE record_index_fields SET state = CASE WHEN field_key IN ($keys) AND kind = 'filter' THEN 'ready' ELSE 'removing' END WHERE tenant_id = $T AND entity_definition_id = $E"
}
search_on() { su "UPDATE record_index_fields SET state = 'ready' WHERE tenant_id = $T AND entity_definition_id = $E AND kind = 'search'"; }
fk() { # fk on|off: the (tenant_id, field_id) -> record_index_fields FK on the four side tables and the term table
    for t in rfv_text rfv_num rfv_ts rfv_bool record_search_terms; do
        if [ "$1" = off ]; then
            su "ALTER TABLE $t DROP CONSTRAINT IF EXISTS ${t}_field_fk"
        else
            su "ALTER TABLE $t ADD CONSTRAINT ${t}_field_fk FOREIGN KEY (tenant_id, field_id) REFERENCES record_index_fields (tenant_id, id)"
        fi
    done
}
run() { # run <label>
    su "CHECKPOINT"
    sh fq.sh write insert "$1" "$SECS" 8
    su "CHECKPOINT"
    sh fq.sh write update "$1" "$SECS" 8
}
F5="tier zone score opened b2"
F10="tier zone site category score opened b001 b025 b2 b40"
# Old FK names from 23_split.sql are rfv_<t>_field_fk; the term table's is record_search_terms_field_fk.
su "ALTER TABLE records DISABLE TRIGGER record_field_values_sync"
run "no-trigger"
su "ALTER TABLE records ENABLE TRIGGER record_field_values_sync"
fields none; run "trigger,0-fields"
fields $F5; run "5-fields,fk"
fields $F10; run "10-fields,fk"
SECS=$SECS sh fq.sh write update_nf "10-fields,fk,unflagged-key-update" "$SECS" 8
search_on; run "10-fields+3-search,fk"
fields $F10
fk off
fields $F5; run "5-fields,no-fk"
fields $F10; run "10-fields,no-fk"
fk on
su "ALTER TABLE records DISABLE TRIGGER record_field_values_sync"
run "no-trigger(repeat)"
su "ALTER TABLE records ENABLE TRIGGER record_field_values_sync"
fields $F10; search_on
echo WRITES-DONE
