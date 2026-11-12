#!/bin/sh
# TICK-714 spike. Runs inside the bench postgres container, like
# bench/records-list/q.sh: nava_app through PgBouncer (pgb), forced RLS,
# explicit tenant_id = $t next to RLS (the P3-01 repository rule).
#   fq.sh prep
#   fq.sh explain <filter> <acl|none> <list|count> <large|small>
#   fq.sh load    <filter> <acl|none> <list|count> <large|small> <seconds> <clients>
#   fq.sh write   <insert|update> <label> <seconds> <clients>
# Filters: /bench/fi/filters.txt (name|predicate on r, pgbench variables).
set -eu
B=/bench/fi
CONN="-h pgb -p 5432 -U nava_app"
APP="$CONN -d nava"
export PGPASSWORD=bench
ACL="(r.requester_id = :me::uuid OR r.assignee_id = :me::uuid OR r.assignment_group_id = ANY (:g::uuid[]) OR EXISTS (SELECT 1 FROM record_shares s WHERE s.tenant_id = r.tenant_id AND s.record_id = r.id AND s.principal_id = :me::uuid)) AND r.state <> 'closed'"

build() { # build <filter> <acl|none> <list|count>
    pred=$(grep "^$1|" "$B/${FILTERS:-s1_filters.txt}" | cut -d'|' -f2-)
    [ -n "$pred" ] || { echo "no filter $1" >&2; exit 2; }
    w="r.tenant_id = :t::uuid AND $pred"
    [ "$2" = acl ] && w="$w AND $ACL"
    case "$1" in u_*)
        # The (d) ACL as a UNION of index-friendly branches drives; the side
        # table is a semi-join. Always ACL'd (the acl argument is ignored).
        sel="SELECT r.* FROM records r"; tail=" ORDER BY r.updated_at DESC, r.id DESC LIMIT 50"
        [ "$3" = count ] && { sel="SELECT count(*) FROM (SELECT 1 FROM records r"; tail=" LIMIT 1001) x"; }
        printf '%s\n' "$sel WHERE r.tenant_id = :t::uuid AND r.state <> 'closed' AND r.id IN (SELECT id FROM records WHERE tenant_id = :t::uuid AND requester_id = :me::uuid AND state <> 'closed' UNION SELECT id FROM records WHERE tenant_id = :t::uuid AND assignee_id = :me::uuid AND state <> 'closed' UNION SELECT id FROM records WHERE tenant_id = :t::uuid AND assignment_group_id = ANY (:g::uuid[]) AND state <> 'closed' UNION SELECT s.record_id FROM record_shares s WHERE s.tenant_id = :t::uuid AND s.principal_id = :me::uuid) AND $pred$tail"
        return ;;
    esac
    case "$1" in d_*)
        # Variant A2: drive from the side table, ordered by its copy of updated_at.
        w="v.tenant_id = :t::uuid AND r.tenant_id = :t::uuid AND $pred"
        [ "$2" = acl ] && w="$w AND $ACL"
        if [ "$3" = list ]; then
            printf '%s\n' "SELECT r.* FROM record_field_values2 v JOIN records r ON r.tenant_id = v.tenant_id AND r.id = v.record_id WHERE $w ORDER BY v.updated_at DESC, v.record_id DESC LIMIT 50"
        else
            printf '%s\n' "SELECT count(*) FROM (SELECT 1 FROM record_field_values2 v JOIN records r ON r.tenant_id = v.tenant_id AND r.id = v.record_id WHERE $w LIMIT 1001) x"
        fi
        return ;;
    esac
    if [ "$3" = list ]; then
        printf '%s\n' "SELECT r.* FROM records r WHERE $w ORDER BY r.updated_at DESC, r.id DESC LIMIT 50"
    else
        printf '%s\n' "SELECT count(*) FROM (SELECT 1 FROM records r WHERE $w LIMIT 1001) x"
    fi
}
actor() { printf '%s\n' "SELECT * FROM bench_actor_f WHERE kind = '$1' AND n = :u"; }

pct() {
    cat /tmp/l/log* | awk '{print $3}' | sort -n | awk '
        {v[NR] = $1}
        function at(p,  i) { i = int(NR * p); if (i < NR * p) i++; if (i < 1) i = 1; return v[i] / 1000 }
        END { printf "tx=%d p50_ms=%.2f p95_ms=%.2f p99_ms=%.2f max_ms=%.2f", NR, at(0.50), at(0.95), at(0.99), v[NR] / 1000 }'
}
run_pgbench() {
    f=$1; secs=$2; c=$3; shift 3
    rm -rf /tmp/l && mkdir -p /tmp/l
    if ! pgbench $CONN -n -M prepared -c "$c" -j "$c" -T "$secs" -l --log-prefix=/tmp/l/log "$@" -f "$f" nava > /tmp/l/out.txt 2>&1; then
        cat /tmp/l/out.txt >&2; exit 1
    fi
    tps=$(sed -n 's/^tps = \([0-9.]*\).*/\1/p' /tmp/l/out.txt | head -n1)
    failed=$(sed -n 's/^number of failed transactions: \([0-9]*\).*/\1/p' /tmp/l/out.txt | head -n1)
    printf 'tps=%s failed=%s %s' "$tps" "${failed:-0}" "$(pct)"
}

case "$1" in
prep)
    psql -X -v ON_ERROR_STOP=1 -q -U pgadmin -d nava <<'EOF'
DROP TABLE IF EXISTS bench_actor_f;
CREATE TABLE bench_actor_f AS
SELECT a.kind, a.n, a.tenant_id AS t, a.user_id AS me, a.groups AS g,
       max(f.field_id::text) FILTER (WHERE f.field_key = 'site') AS f_site,
       max(f.field_id::text) FILTER (WHERE f.field_key = 'priority') AS f_priority,
       max(f.field_id::text) FILTER (WHERE f.field_key = 'category') AS f_category,
       max(f.field_id::text) FILTER (WHERE f.field_key = 'due_at') AS f_due_at,
       max(f.field_id::text) FILTER (WHERE f.field_key = 'vip') AS f_vip,
       max(f.field_id::text) FILTER (WHERE f.field_key = 'impact') AS f_impact
FROM bench_actor a JOIN bench_field f ON f.tenant_id = a.tenant_id
GROUP BY 1, 2, 3, 4, 5;
ALTER TABLE bench_actor_f ADD PRIMARY KEY (kind, n);
GRANT SELECT ON bench_actor_f TO nava_app;
EOF
    ;;
explain)
    f=$2 acl=$3 mode=$4 kind=$5
    sql=$(build "$f" "$acl" "$mode" | sed "s/:\([a-z_0-9]*\)::/:'\1'::/g")
    echo "--- EXPLAIN filter=$f acl=$acl mode=$mode tenant=$kind (actor 1, custom plan)"
    psql -X -v ON_ERROR_STOP=1 -q $APP <<EOF
$(actor "$kind" | sed 's/:u$/1/') \gset
BEGIN;
SELECT set_config('app.tenant_id', :'t', true) \g /dev/null
EXPLAIN (ANALYZE, BUFFERS) $sql;
COMMIT;
EOF
    ;;
load)
    f=$2 acl=$3 mode=$4 kind=$5 secs=$6 c=$7
    n=$(psql -X -At $APP -c "SELECT count(*) FROM bench_actor_f WHERE kind = '$kind'")
    s=/tmp/fscript.sql
    {
        printf '%s\n' '\set u random(1, :nact)'
        printf '%s \\gset\n' "$(actor "$kind")"
        printf '%s\n' 'BEGIN;'
        printf '%s\n' "SELECT set_config('app.tenant_id', :t::text, true);"
        printf '%s;\n' "$(build "$f" "$acl" "$mode")"
        printf '%s\n' 'COMMIT;'
    } > "$s"
    plan=$(psql -X -At $APP -c "SHOW plan_cache_mode")
    echo "RESULT plan=$plan filter=$f acl=$acl mode=$mode tenant=$kind actors=$n $(run_pgbench "$s" "$secs" "$c" -D nact="$n")"
    ;;
write)
    kind=$2 label=$3 secs=$4 c=$5
    t=$(psql -X -At -U pgadmin -d nava -c "SELECT md5('tenant0')::uuid")
    s=/tmp/write_$kind.sql
    if [ "$kind" = insert ]; then
        # As bench/records-list/q.sh write insert, in the S1 shape (ten
        # filterable fields, three free-text fields), into the 400k type.
        cat > "$s" <<'EOF'
\set n random(1, 5000)
\set g random(0, 49)
\set h random(0, 99999)
BEGIN;
SELECT set_config('app.tenant_id', :t::text, true);
INSERT INTO records (tenant_id, id, entity_definition_id, number, state, requester_id, assignee_id, assignment_group_id, created_at, updated_at, data) VALUES (:t::uuid, gen_random_uuid(), md5('ed0/0')::uuid, 0, 'new', md5('u0/' || :n)::uuid, md5('u0/' || :n)::uuid, md5('g0/' || :g)::uuid, now(), now(), jsonb_build_object('priority', (1 + :g % 5)::text, 'category', 'category-' || :g, 'impact', 2, 'title', 'Bench insert ' || md5(random()::text), 'description', repeat(md5(random()::text), 27), 'tags', jsonb_build_array('t1', 't2'), 'site', 's' || :g, 'tier', 'z' || :g, 'zone', 'q' || :g, 'score', :h / 10.0, 'opened', '2026-11-11T10:00:00Z', 'b001', false, 'b025', false, 'b2', :g = 1, 'b40', :g < 20, 'summary', bench_words(:h || 's' || random(), 12), 'notes', bench_words(:h || 'n' || random(), 40)));
COMMIT;
EOF
    elif [ "$kind" = update ]; then
        # As q.sh write update, but the data key changed is a filterable one (score).
        cat > "$s" <<'EOF'
\set n random(1, 400000)
BEGIN;
SELECT set_config('app.tenant_id', :t::text, true);
UPDATE records SET updated_at = now(), state = CASE WHEN state = 'closed' THEN 'new' ELSE 'closed' END, data = jsonb_set(data, '{score}', to_jsonb(:n % 10000)) WHERE tenant_id = :t::uuid AND id = md5('r0/' || :n)::uuid;
COMMIT;
EOF
    else
        # update_nf: exactly q.sh write update (changes impact, not a flagged field).
        cat > "$s" <<'EOF'
\set n random(1, 400000)
BEGIN;
SELECT set_config('app.tenant_id', :t::text, true);
UPDATE records SET updated_at = now(), state = CASE WHEN state = 'closed' THEN 'new' ELSE 'closed' END, data = jsonb_set(data, '{impact}', to_jsonb(:n % 3 + 1)) WHERE tenant_id = :t::uuid AND id = md5('r0/' || :n)::uuid;
COMMIT;
EOF
    fi
    echo "WRITE kind=$kind set=$label $(run_pgbench "$s" "$secs" "$c" -D t="$t")"
    ;;
*) echo "usage" >&2; exit 2 ;;
esac
