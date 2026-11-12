#!/bin/sh
# TICK-714 S3 deterministic cases (TA R1, R3) on the RFC rev 1 schema, run
# inside the bench container on small tenant 1 (type ed1/0). Timing is fixed
# with pg_sleep so the interleaving does not depend on luck.
#   races.sh r1 <barrier|nobarrier>   live INSERT whose trigger runs before the flip
#   races.sh r3 <barrier|nobarrier>   live UPDATE whose trigger saw ready, across the removal
set -eu
export PGPASSWORD=bench
APP="-X -q -At -v ON_ERROR_STOP=1 -h localhost -U nava_app -d nava"
T=$(psql -X -At -U pgadmin -d nava -c "SELECT md5('tenant1')::uuid")
E=$(psql -X -At -U pgadmin -d nava -c "SELECT md5('ed1/0')::uuid")
S=/bench/fi/s3.sh
# Each run uses its own field key so runs do not interact.
K=race_$(date +%s%N)

live_insert() { # trigger runs at t=0, COMMIT at t=4 s
    psql $APP <<EOF
BEGIN;
SELECT set_config('app.tenant_id', '$T', true) \g /dev/null
INSERT INTO records (tenant_id, id, entity_definition_id, number, state, requester_id, created_at, updated_at, data)
VALUES ('$T', md5('$K')::uuid, '$E', 0, 'new', md5('u1/1')::uuid, now(), now(), jsonb_build_object('$K', 'live'));
SELECT pg_sleep(4) \g /dev/null
COMMIT;
EOF
}
live_update() { # trigger runs at t=0 (field ready), COMMIT at t=4 s
    psql $APP <<EOF
BEGIN;
SELECT set_config('app.tenant_id', '$T', true) \g /dev/null
UPDATE records SET data = data || jsonb_build_object('$K', 'changed')
WHERE tenant_id = '$T' AND id = (SELECT id FROM records WHERE tenant_id = '$T' AND entity_definition_id = '$E' ORDER BY id DESC LIMIT 1);
SELECT pg_sleep(4) \g /dev/null
COMMIT;
EOF
}

# Every record of the type gets a value for the race field (as superuser,
# no trigger copies yet: the field is not registered).
psql -X -q -U pgadmin -d nava -c "UPDATE records SET data = data || jsonb_build_object('$K', 'v' || (number % 10)) WHERE tenant_id = '$T' AND entity_definition_id = '$E'"

case "$1" in
r1)
    live_insert & lp=$!
    sleep 1
    f=$(sh $S flip "$T" "$E" "$K" text)
    sh $S backfill "$T" "$f" 0 "$2" >/dev/null
    wait $lp
    echo "R1 ($2): $(sh $S diff "$T" "$f")"
    ;;
r3)
    # The target record has no value for the field, so the live writer adds
    # a NEW side row (one that the removal's DELETE cannot block on).
    psql -X -q -U pgadmin -d nava -c "UPDATE records SET data = data - '$K' WHERE tenant_id = '$T' AND id = (SELECT id FROM records WHERE tenant_id = '$T' AND entity_definition_id = '$E' ORDER BY id DESC LIMIT 1)"
    f=$(sh $S flip "$T" "$E" "$K" text)
    sh $S backfill "$T" "$f" 0 "$2" >/dev/null
    live_update & lp=$!
    sleep 1
    res=$(sh $S remove "$T" "$f" "$2" 2>&1 || true)
    wait $lp
    orphans=$(psql -X -At -U pgadmin -d nava -c "SELECT count(*) FROM rfv_text v WHERE v.tenant_id = '$T' AND v.field_id = '$f'")
    reg=$(psql -X -At -U pgadmin -d nava -c "SELECT coalesce(max(state), 'deleted') FROM record_index_fields WHERE tenant_id = '$T' AND id = '$f'")
    echo "R3 ($2): $res; side rows left for the field: $orphans; registry row: $reg"
    ;;
esac
