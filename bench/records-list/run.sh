#!/bin/sh
# TICK-339 (P3-01, the PLAN-003 D25 question): an ACL-filtered, sorted,
# keyset-paged list over 800k synthetic records, as nava_app with forced RLS,
# through PgBouncer (transaction mode), loaded with pgbench. Report:
# docs/data/records-list-bench.md. Output: bin/records-list-bench-output.txt
# (gitignored). Needs Docker only. Not wired into CI.
#   make bench-records-list            # everything, about 140 minutes
#   sh bench/records-list/run.sh [step...]
# Steps (default: all, in this order): setup base expr gin custom base expr
# gin writes. The first base/expr/gin pass runs with plan_cache_mode = auto
# (the default: a cached prepared statement may switch to a generic plan),
# the second with force_custom_plan for nava_app.
# Env: BENCH_SECONDS (60) per read run, BENCH_WRITE_SECONDS (60), BENCH_CLIENTS
# (8), BENCH_ID (the container suffix; set it with BENCH_KEEP=1 to reuse a
# seeded database across invocations), BENCH_OUT (the output file).
set -eu
cd "$(dirname "$0")/../.."
dir=bench/records-list
pgtag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
pbtag=$(sed -n 's/^| `edoburu\/pgbouncer` | \([^ |]*\) |.*/\1/p' docs/pins.md)
id=${BENCH_ID:-$$}
net=nava-rlb-$id
pg=nava-rlb-pg-$id
pb=nava-rlb-pgb-$id
out=${BENCH_OUT:-bin/records-list-bench-output.txt}
secs=${BENCH_SECONDS:-60}
wsecs=${BENCH_WRITE_SECONDS:-$secs}
clients=${BENCH_CLIENTS:-8}
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

cleanup() {
    if [ "${BENCH_KEEP:-0}" = 1 ]; then
        echo "kept: $pg $pb (network $net)"
        return
    fi
    docker rm -f "$pg" "$pb" >/dev/null 2>&1 || true
    docker network rm "$net" >/dev/null 2>&1 || true
}
trap cleanup EXIT

su() { docker exec -i "$pg" psql -X -v ON_ERROR_STOP=1 -U pgadmin -d "${2:-nava}" -f "/bench/$1"; }
susql() { docker exec -i "$pg" psql -X -v ON_ERROR_STOP=1 -U pgadmin -d nava -c "$1"; }
q() { docker exec -i "$pg" sh /bench/q.sh "$@"; }

setup() {
    docker network create "$net" >/dev/null
    # PostgreSQL defaults except memory: the table and indexes fit in
    # shared_buffers (warm cache). fsync stays on, so write numbers are real.
    docker run -d --name "$pg" --network "$net" --shm-size=3g \
        -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench "postgres:$pgtag" \
        -c shared_buffers=2GB -c effective_cache_size=6GB -c max_wal_size=8GB \
        -c track_io_timing=on >/dev/null
    until docker exec "$pg" pg_isready -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
    sleep 2
    until docker exec "$pg" pg_isready -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
    docker cp "$dir/." "$pg:/bench"
    su 00_roles.sql postgres
    su 01_schema.sql
    sed "s/PGHOST/$pg/" "$dir/pgbouncer.ini" > "bin/records-list-pgbouncer.ini"
    printf '"nava_app" "bench"\n' > bin/records-list-userlist.txt
    docker create --name "$pb" --network "$net" --network-alias pgb "edoburu/pgbouncer:$pbtag" >/dev/null
    docker cp bin/records-list-pgbouncer.ini "$pb:/etc/pgbouncer/pgbouncer.ini"
    docker cp bin/records-list-userlist.txt "$pb:/etc/pgbouncer/userlist.txt"
    docker start "$pb" >/dev/null
    echo "### seed"
    su 02_seed.sql
    echo "### index set (proposed)"
    su 03_indexes.sql
    su 04_p20.sql
    for x in a b c c2 d; do q cursors "$x"; done
    echo "### sizes, proposed set"
    su 05_sizes.sql
}

reads() { # reads <q...>: list (page 1 and 20) and the capped count, RLS alone and explicit tenant_id
    su 06_prewarm.sql >/dev/null
    for x in "$@"; do
        for p in rls explicit; do
            for pg_ in 1 20; do q explain "$x" list large "$p" "$pg_"; done
            q explain "$x" list small "$p" 1
            q explain "$x" count large "$p" 1
            q explain "$x" count small "$p" 1
        done
    done
    for x in "$@"; do
        for p in rls explicit; do
            q load "$x" list large "$p" 1 "$secs" "$clients"
            q load "$x" list large "$p" 20 "$secs" "$clients"
            q load "$x" list small "$p" 1 "$secs" "$clients"
            echo "RESULT q=$x mode=list tenant=small pred=$p page=20 n/a (a small tenant has about 1,000 records; no list reaches page 20)"
            q load "$x" count large "$p" 1 "$secs" "$clients"
            q load "$x" count small "$p" 1 "$secs" "$clients"
        done
    done
}

base() {
    echo "### reads, proposed set: a b c(no expression index) d"
    reads d a b c
}

expr() {
    echo "### (c) with an expression index"
    susql "SET maintenance_work_mem = '512MB'; SET ROLE bench_owner; CREATE INDEX records_tenant_priority ON records (tenant_id, (data ->> 'priority'), updated_at DESC, id DESC); RESET ROLE; ANALYZE records;"
    su 05_sizes.sql
    reads c
    susql "DROP INDEX records_tenant_priority; ANALYZE records;"
}

gin() {
    echo "### (c2) one jsonb_path_ops GIN on data"
    susql "SET maintenance_work_mem = '512MB'; SET ROLE bench_owner; CREATE INDEX records_data_gin ON records USING gin (data jsonb_path_ops); RESET ROLE; ANALYZE records;"
    su 05_sizes.sql
    reads c2
    susql "DROP INDEX records_data_gin; ANALYZE records;"
}

custom() {
    # Every later read runs with custom plans: nava_app is re-planned for each
    # execution's parameters. PgBouncer is restarted so its server
    # connections start with the role setting.
    echo "### plan_cache_mode = force_custom_plan for nava_app"
    susql "ALTER ROLE nava_app SET plan_cache_mode = force_custom_plan;"
    docker restart "$pb" >/dev/null
    until docker exec "$pg" sh -c 'PGPASSWORD=bench psql -X -At -h pgb -U nava_app -d nava -c "SHOW plan_cache_mode"' 2>/dev/null | grep -q force_custom_plan; do sleep 1; done
}

writes() {
    echo "### writes (8 clients, large tenant, nava_app through PgBouncer)"
    has_gin=$(docker exec "$pg" psql -X -At -U pgadmin -d nava -c "SELECT count(*) FROM pg_class WHERE relname = 'records_data_gin'")
    if [ "$has_gin" = 0 ]; then
        susql "SET maintenance_work_mem = '512MB'; SET ROLE bench_owner; CREATE INDEX records_data_gin ON records USING gin (data jsonb_path_ops); RESET ROLE;"
    fi
    susql "CHECKPOINT;"
    q write insert "proposed+gin(fastupdate=on)" "$wsecs" "$clients"
    susql "SELECT version, pending_pages, pending_tuples FROM pgstatginindex('records_data_gin');"
    q write update "proposed+gin(fastupdate=on)" "$wsecs" "$clients"
    susql "SELECT version, pending_pages, pending_tuples FROM pgstatginindex('records_data_gin');"
    susql "ALTER INDEX records_data_gin SET (fastupdate = off); SELECT gin_clean_pending_list('records_data_gin'); CHECKPOINT;"
    q write insert "proposed+gin(fastupdate=off)" "$wsecs" "$clients"
    q write update "proposed+gin(fastupdate=off)" "$wsecs" "$clients"
    susql "DROP INDEX records_data_gin; CHECKPOINT;"
    q write insert "proposed" "$wsecs" "$clients"
    q write update "proposed" "$wsecs" "$clients"
    susql "DROP INDEX records_tenant_updated, records_tenant_requester, records_tenant_assignee_open, records_tenant_group_open; CHECKPOINT;"
    q write insert "pkey-only" "$wsecs" "$clients"
    q write update "pkey-only" "$wsecs" "$clients"
}

[ $# -eq 0 ] && set -- setup base expr gin custom base expr gin writes
mkdir -p bin "$(dirname "$out")"
{
    echo "### $(date -u +%Y-%m-%dT%H:%M:%SZ) steps: $*; ${secs}s reads, ${wsecs}s writes, $clients clients"
    echo "### host: $(uname -s); docker $(docker version --format '{{.Server.Version}}')"
    docker info --format '### docker VM: {{.NCPU}} CPUs, {{.MemTotal}} bytes, kernel {{.KernelVersion}}'
    echo "### postgres:$pgtag, edoburu/pgbouncer:$pbtag"
    for s in "$@"; do
        "$s"
        [ "$s" = setup ] && docker exec "$pg" psql -X -U pgadmin -d nava -c "SELECT version();" \
            -c "SELECT name, setting, unit FROM pg_settings WHERE name IN ('shared_buffers','effective_cache_size','work_mem','random_page_cost','max_parallel_workers_per_gather','jit','plan_cache_mode','max_connections','fsync','synchronous_commit') ORDER BY name;"
    done
} 2>&1 | tee -a "$out"
echo "wrote $out"
