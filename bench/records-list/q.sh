#!/bin/sh
# Runs inside the bench postgres container (it has psql and pgbench). Every
# query is built from one predicate template, bench/records-list/pred_<q>.sql,
# written with pgbench parameters (:me::uuid, :g::uuid[]); the EXPLAIN and the
# page-20 cursor forms are derived from the same text.
#   q.sh load    <q> <list|count> <large|small> <rls|explicit> <1|20> <seconds> <clients>
#   q.sh explain <q> <list|count> <large|small> <rls|explicit> <1|20>
#   q.sh cursors <q>
#   q.sh write   <insert|update> <label> <seconds> <clients>
set -eu
B=/bench
CONN="-h pgb -p 5432 -U nava_app"
APP="$CONN -d nava"
export PGPASSWORD=bench

build() { # build <q> <list|count> <rls|explicit> <1|20>: the SQL, pgbench form
    pred=$(cat "$B/pred_$1.sql")
    ten=""
    [ "$3" = explicit ] && ten=" AND r.tenant_id = :t::uuid"
    cur=""
    [ "$4" = 20 ] && cur=" AND (r.updated_at, r.id) < (:cts::timestamptz, :cid::uuid)"
    if [ "$2" = list ]; then
        printf '%s\n' "SELECT r.* FROM records r WHERE $pred$ten$cur ORDER BY r.updated_at DESC, r.id DESC LIMIT 50"
    else
        # The capped count (PLAN-003): reported separately from the list.
        printf '%s\n' "SELECT count(*) FROM (SELECT 1 FROM records r WHERE $pred$ten LIMIT 1001) x"
    fi
}

actor() { # actor <q> <large|small> <1|20>: the SELECT that \gsets t, me, g (and cts, cid)
    if [ "$3" = 20 ]; then
        printf '%s\n' "SELECT tenant_id AS t, user_id AS me, groups AS g, cts, cid FROM bench_p20 WHERE q = '$1' AND n = :u"
    else
        printf '%s\n' "SELECT tenant_id AS t, user_id AS me, groups AS g FROM bench_actor WHERE kind = '$2' AND n = :u"
    fi
}

nactors() { # nactors <q> <large|small> <1|20>
    if [ "$3" = 20 ]; then
        psql -X -At $APP -c "SELECT count(*) FROM bench_p20 WHERE q = '$1'"
    elif [ "$2" = large ]; then
        psql -X -At $APP -c "SELECT count(*) FROM bench_actor WHERE kind = 'large'"
    else
        psql -X -At $APP -c "SELECT count(*) FROM bench_actor WHERE kind = 'small'"
    fi
}

pct() { # latency percentiles (ms) from pgbench -l logs: field 3 is the latency in us
    cat /tmp/l/log* | awk '{print $3}' | sort -n | awk '
        {v[NR] = $1}
        function at(p,  i) { i = int(NR * p); if (i < NR * p) i++; if (i < 1) i = 1; return v[i] / 1000 }
        END { printf "tx=%d p50_ms=%.2f p95_ms=%.2f p99_ms=%.2f max_ms=%.2f", NR, at(0.50), at(0.95), at(0.99), v[NR] / 1000 }'
}

run_pgbench() { # run_pgbench <script> <seconds> <clients> <extra args...>
    f=$1; secs=$2; c=$3; shift 3
    rm -rf /tmp/l && mkdir -p /tmp/l
    if ! pgbench $CONN -n -M prepared -c "$c" -j "$c" -T "$secs" -l --log-prefix=/tmp/l/log "$@" -f "$f" nava > /tmp/l/out.txt 2>&1; then
        cat /tmp/l/out.txt >&2
        exit 1
    fi
    tps=$(sed -n 's/^tps = \([0-9.]*\).*/\1/p' /tmp/l/out.txt | head -n1)
    failed=$(sed -n 's/^number of failed transactions: \([0-9]*\).*/\1/p' /tmp/l/out.txt | head -n1)
    printf 'tps=%s failed=%s %s' "$tps" "${failed:-0}" "$(pct)"
}

case "$1" in
load)
    q=$2 mode=$3 kind=$4 pred=$5 page=$6 secs=$7 c=$8
    n=$(nactors "$q" "$kind" "$page")
    if [ "$n" -eq 0 ]; then
        echo "RESULT q=$q mode=$mode tenant=$kind pred=$pred page=$page n/a (no actor has a page $page)"
        exit 0
    fi
    f=/tmp/script_$q.sql
    {
        printf '%s\n' '\set u random(1, :nact)'
        printf '%s \\gset\n' "$(actor "$q" "$kind" "$page")"
        printf '%s\n' 'BEGIN;'
        printf '%s\n' "SELECT set_config('app.tenant_id', :t::text, true);"
        printf '%s;\n' "$(build "$q" "$mode" "$pred" "$page")"
        printf '%s\n' 'COMMIT;'
    } > "$f"
    plan=$(psql -X -At $APP -c "SHOW plan_cache_mode")
    echo "RESULT plan=$plan q=$q mode=$mode tenant=$kind pred=$pred page=$page actors=$n $(run_pgbench "$f" "$secs" "$c" -D nact="$n")"
    ;;
explain)
    q=$2 mode=$3 kind=$4 pred=$5 page=$6
    sql=$(build "$q" "$mode" "$pred" "$page" | sed "s/:\(me\|g\|t\|cts\|cid\)::/:'\1'::/g")
    a=$(actor "$q" "$kind" "$page" | sed 's/:u$/1/')
    echo "--- EXPLAIN q=$q mode=$mode tenant=$kind pred=$pred page=$page (actor 1)"
    if [ "$(psql -X -At $APP -c "SELECT count(*) FROM ($a) x")" -eq 0 ]; then
        echo "n/a (no actor has a page $page)"
        exit 0
    fi
    psql -X -v ON_ERROR_STOP=1 -q $APP <<EOF
$a \gset
BEGIN;
SELECT set_config('app.tenant_id', :'t', true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) $sql;
COMMIT;
EOF
    # The plan a cached prepared statement (pgbench -M prepared, pgx's
    # statement cache) switches to under plan_cache_mode = auto: the generic
    # plan, for the same query and actor.
    gsql=$(build "$q" "$mode" "$pred" "$page" | sed 's/:me::/$1::/g; s/:g::/$2::/g; s/:t::/$3::/g; s/:cts::/$4::/g; s/:cid::/$5::/g')
    args="NULL, NULL"
    [ "$page" = 20 ] && args=":'cts', :'cid'"
    echo "--- generic plan, same query and actor"
    psql -X -v ON_ERROR_STOP=1 -q $APP <<EOF
$a \gset
BEGIN;
SELECT set_config('app.tenant_id', :'t', true) \g /dev/null
SET LOCAL plan_cache_mode = force_generic_plan;
PREPARE p(uuid, uuid[], uuid, timestamptz, uuid) AS $gsql;
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) EXECUTE p(:'me', :'g', :'t', $args);
DEALLOCATE p;
COMMIT;
EOF
    ;;
cursors)
    # The cursor of page 20 is the 950th row (the last of page 19), computed
    # once, as the superuser, from the same predicate.
    q=$2
    pred=$(sed 's/:me::uuid/a.user_id/g; s/:g::uuid\[\]/a.groups/g' "$B/pred_$q.sql")
    if [ "$q" = b ]; then
        src="SELECT n, tenant_id, user_id, '{}'::uuid[] AS groups FROM bench_heavy"
    else
        src="SELECT n, tenant_id, user_id, groups FROM bench_actor WHERE kind = 'large'"
    fi
    psql -X -v ON_ERROR_STOP=1 -q -U pgadmin -d nava <<EOF
INSERT INTO bench_p20 (q, n, tenant_id, user_id, groups, cts, cid)
SELECT '$q', row_number() OVER (ORDER BY a.n), a.tenant_id, a.user_id, a.groups, c.updated_at, c.id
FROM ($src) a
CROSS JOIN LATERAL (
    SELECT r.updated_at, r.id FROM records r
    WHERE r.tenant_id = a.tenant_id AND $pred
    ORDER BY r.updated_at DESC, r.id DESC OFFSET 949 LIMIT 1) c;
SELECT '$q' AS q, count(*) AS page20_actors FROM bench_p20 WHERE q = '$q';
EOF
    ;;
write)
    # Write throughput as nava_app through PgBouncer, into the large tenant.
    kind=$2 label=$3 secs=$4 c=$5
    t=$(psql -X -At -U pgadmin -d nava -c "SELECT md5('tenant0')::uuid")
    f=/tmp/write_$kind.sql
    if [ "$kind" = insert ]; then
        cat > "$f" <<'EOF'
\set n random(1, 5000)
\set g random(0, 49)
BEGIN;
SELECT set_config('app.tenant_id', :t::text, true);
INSERT INTO records (tenant_id, id, entity_definition_id, number, state, requester_id, assignee_id, assignment_group_id, created_at, updated_at, data) VALUES (:t::uuid, gen_random_uuid(), md5('ed0/1')::uuid, 0, 'new', md5('u0/' || :n)::uuid, md5('u0/' || :n)::uuid, md5('g0/' || :g)::uuid, now(), now(), jsonb_build_object('priority', (1 + :g % 5)::text, 'category', 'category-' || :g, 'impact', 2, 'title', 'Bench insert ' || md5(random()::text), 'description', repeat(md5(random()::text), 27), 'tags', jsonb_build_array('t1', 't2')));
COMMIT;
EOF
    else
        # Moves updated_at and the state: every list index on updated_at changes.
        cat > "$f" <<'EOF'
\set n random(1, 400000)
BEGIN;
SELECT set_config('app.tenant_id', :t::text, true);
UPDATE records SET updated_at = now(), state = CASE WHEN state = 'closed' THEN 'new' ELSE 'closed' END, data = jsonb_set(data, '{impact}', to_jsonb(:n % 3 + 1)) WHERE tenant_id = :t::uuid AND id = md5('r0/' || :n)::uuid;
COMMIT;
EOF
    fi
    echo "WRITE kind=$kind set=$label $(run_pgbench "$f" "$secs" "$c" -D t="$t")"
    ;;
*)
    echo "usage: q.sh load|explain|cursors|write ..." >&2
    exit 2
    ;;
esac
