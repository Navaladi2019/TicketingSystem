#!/bin/sh
# TICK-714 S3 (RFC-015 rev 1 §5, D10, D11, D17). Runs inside the bench
# postgres container. The Temporal workflow is played by this shell: each
# psql call below is one activity, in one transaction, as nava_app under
# forced RLS (direct to postgres, not through PgBouncer, like a worker).
#   s3.sh flip     <tenant> <type> <field_key> <value_type>   -> prints field id
#   s3.sh barrier                                             (D17)
#   s3.sh backfill <tenant> <field_id> [pause_ms] [nobarrier] (D10 + D17)
#   s3.sh remove   <tenant> <field_id> [nobarrier]            (D11 + D17)
#   s3.sh diff     <tenant> <field_id>                        (superuser)
set -eu
export PGPASSWORD=bench
APP="-X -q -At -v ON_ERROR_STOP=1 -h localhost -U nava_app -d nava"
SU="-X -q -At -v ON_ERROR_STOP=1 -U pgadmin -d nava"
appc() { printf "%s
" "$1" | psql $APP; }
now_ms() { date +%s%3N; }

barrier() {
    # One snapshot after the flip committed; then poll pg_xact_status once
    # per "activity attempt", with a 1 s "workflow timer" between polls.
    xips=$(appc "SELECT coalesce(string_agg(x::text, ','), '') FROM pg_snapshot_xip(pg_current_snapshot()) x")
    t0=$(now_ms); polls=0
    while :; do
        polls=$((polls + 1))
        [ -z "$xips" ] && break
        left=$(appc "SELECT count(*) FROM unnest('{$xips}'::xid8[]) x WHERE pg_xact_status(x) = 'in progress'")
        [ "$left" -eq 0 ] && break
        sleep 1
    done
    echo "BARRIER xids=[${xips}] polls=$polls waited_ms=$(( $(now_ms) - t0 ))" >&2
}

case "$1" in
flip)
    t=$2 e=$3 k=$4 vt=$5
    psql $APP <<EOF
BEGIN;
SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, kind, state)
VALUES ('$t', md5('s3' || '$t' || '$e' || '$k' || clock_timestamp())::uuid, '$e', '$k', '$vt', 'filter', 'backfilling')
RETURNING id;
COMMIT;
EOF
    ;;
barrier) barrier ;;
backfill)
    t=$2 f=$3 pause=${4:-200} mode=${5:-barrier}
    [ "$mode" = barrier ] && barrier
    cursor=00000000-0000-0000-0000-000000000000
    total=$(appc "BEGIN; SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
SELECT count(*) FROM records r JOIN record_index_fields f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id WHERE f.id = '$f';
COMMIT;")
    done_=0; batches=0; t0=$(now_ms)
    while :; do
        b0=$(now_ms)
        out=$(psql $APP <<EOF
BEGIN;
SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
SELECT pg_current_wal_lsn() AS lsn0 \gset
-- (a) the registry row FOR SHARE: stop unless backfilling
SELECT state AS st, entity_definition_id AS e, field_key AS k, value_type AS vt
FROM record_index_fields WHERE tenant_id = '$t' AND id = '$f' FOR SHARE \gset
\if :{?st}
\else
\set st gone
\endif
SELECT :'st' = 'backfilling' AS go \gset
\if :go
-- (b) the next 1,000 records, locked FOR SHARE (latest committed data)
SELECT coalesce(array_agg(id ORDER BY id), '{}') AS ids, count(*) AS n, coalesce(max(id::text), '') AS hi
FROM (SELECT id FROM records WHERE tenant_id = '$t' AND entity_definition_id = :'e'
      AND id > '$cursor' ORDER BY id LIMIT 1000 FOR SHARE) b \gset
-- (c) per (record, field): DELETE, then INSERT the D19-coerced value
DELETE FROM rfv_text WHERE tenant_id = '$t' AND field_id = '$f' AND record_id = ANY (:'ids'::uuid[]);
INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value)
SELECT r.tenant_id, '$f', r.id, 0, c.value_text
FROM records r CROSS JOIN LATERAL record_field_coerce(r.data -> :'k', :'vt') c
WHERE r.tenant_id = '$t' AND r.id = ANY (:'ids'::uuid[])
  AND c.value_text IS NOT NULL;  -- S3 backfills text fields: rfv_text only
-- (d) progress, a whole percentage
UPDATE record_index_fields SET progress = least(99, (($done_ + :n) * 100 / greatest($total, 1))), updated_at = now()
WHERE tenant_id = '$t' AND id = '$f';
SELECT :n || ' ' || coalesce(:'hi', '') || ' ' || pg_wal_lsn_diff(pg_current_wal_lsn(), :'lsn0');
\else
SELECT 'stop ' || :'st';
\endif
COMMIT;
EOF
)
        set -- $out
        [ "$1" = stop ] && { echo "BACKFILL stopped: state $2" >&2; exit 0; }
        n=$1
        [ "$n" -eq 0 ] && break
        batches=$((batches + 1)); done_=$((done_ + n)); cursor=$2
        echo "BATCH $batches rows=$n ms=$(( $(now_ms) - b0 )) wal_bytes=$3"
        sleep "$(awk "BEGIN{print $pause/1000}")"
    done
    appc "BEGIN; SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
UPDATE record_index_fields SET state = 'ready', progress = 100, updated_at = now() WHERE tenant_id = '$t' AND id = '$f' AND state = 'backfilling'; COMMIT;"
    [ "$mode" = barrier ] && barrier
    echo "BACKFILL done rows=$done_ batches=$batches ms=$(( $(now_ms) - t0 ))"
    ;;
remove)
    t=$2 f=$3 mode=${4:-barrier}
    appc "BEGIN; SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
UPDATE record_index_fields SET state = 'removing', updated_at = now() WHERE tenant_id = '$t' AND id = '$f'; COMMIT;"
    [ "$mode" = barrier ] && barrier
    while :; do
        while :; do
            n=$(appc "BEGIN; SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
WITH d AS (DELETE FROM rfv_text WHERE ctid = ANY (ARRAY(SELECT ctid FROM rfv_text WHERE tenant_id = '$t' AND field_id = '$f' LIMIT 5000)) RETURNING 1) SELECT count(*) FROM d; COMMIT;")
            [ "$n" -eq 0 ] && break
        done
        left=$(appc "BEGIN; SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
SELECT count(*) FROM (SELECT 1 FROM rfv_text WHERE tenant_id = '$t' AND field_id = '$f' LIMIT 1) x; COMMIT;")
        [ "$left" -eq 0 ] && break
    done
    if appc "BEGIN; SELECT set_config('app.tenant_id', '$t', true) \g /dev/null
DELETE FROM record_index_fields WHERE tenant_id = '$t' AND id = '$f'; COMMIT;" 2>/tmp/rm.err; then
        echo "REMOVE registry row deleted"
    else
        echo "REMOVE registry delete FAILED: $(head -c 300 /tmp/rm.err)"
    fi
    ;;
diff)
    t=$2 f=$3
    psql $SU <<EOF
WITH f AS (SELECT * FROM record_index_fields WHERE tenant_id = '$t' AND id = '$f'),
exp AS (
    SELECT r.id AS record_id, c.value_text AS value
    FROM records r JOIN f ON f.tenant_id = r.tenant_id AND f.entity_definition_id = r.entity_definition_id
    CROSS JOIN LATERAL record_field_coerce(r.data -> f.field_key, f.value_type) c
    WHERE c.value_text IS NOT NULL),
act AS (SELECT record_id, value FROM rfv_text WHERE tenant_id = '$t' AND field_id = '$f')
SELECT 'DIFF expected=' || (SELECT count(*) FROM exp) || ' actual=' || (SELECT count(*) FROM act)
    || ' missing_or_stale=' || (SELECT count(*) FROM (SELECT * FROM exp EXCEPT ALL SELECT * FROM act) x)
    || ' extra=' || (SELECT count(*) FROM (SELECT * FROM act EXCEPT ALL SELECT * FROM exp) y)
    || ' orphans_any_field=' || (SELECT count(*) FROM rfv_text v WHERE v.tenant_id = '$t'
                                  AND NOT EXISTS (SELECT 1 FROM record_index_fields g WHERE g.tenant_id = v.tenant_id AND g.id = v.field_id));
EOF
    ;;
esac
