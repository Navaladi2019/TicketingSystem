#!/bin/sh
# T13 follow-up (TICK-146) purge-index benchmark: a throwaway postgres from
# docs/pins.md, db-roles and every migration through nava-db, the seed in
# 01_seed.sql, then the purge plans as nava_purge with RLS on, without the
# 00005 indexes and with them (built CONCURRENTLY over the seeded tables).
# synchronize_seqscans is off so a scan does not start where the last one
# stopped; full_page_writes is off so 04_writes.sql measures records only.
# Last, 06_backlog.sql expires half of every table and the plans run again.
# Output in bin/purge-bench-output.txt (gitignored). Needs Docker and Go.
#   sh docs/data/purge-bench/run.sh
set -eu
cd "$(dirname "$0")/../../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="nava-t146-bench-$$"
dir=docs/data/purge-bench
out=bin/purge-bench-output.txt
bin=bin/purge-bench-nava-db$(go env GOEXE)
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -f "$bin"' EXIT

mkdir -p bin
go build -o "$bin" ./cmd/nava-db
docker run -d --name "$name" --shm-size=1g -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench \
    -p 127.0.0.1::5432 "postgres:$tag" -c shared_buffers=1GB \
    -c synchronize_seqscans=off -c full_page_writes=off >/dev/null
until docker exec "$name" pg_isready -h 127.0.0.1 -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
port=$(docker port "$name" 5432/tcp | sed 's/.*://' | head -n1)

for r in nava_migrator nava_app nava_worker nava_relay nava_purge nava_onboard nava_groupsync nava_backup keycloak temporal pgbouncer_auth; do
    export "DBROLES_PASSWORD_$(echo "$r" | tr '[:lower:]' '[:upper:]')=bench-$r"
done
PGHOST=127.0.0.1 PGPORT=$port PGUSER=pgadmin PGPASSWORD=bench PGDATABASE=postgres "$bin" roles >/dev/null
PGHOST=127.0.0.1 PGPORT=$port PGUSER=nava_migrator PGPASSWORD=bench-nava_migrator PGDATABASE=nava "$bin" migrate >/dev/null

psql_as() { # psql_as <role> <file>
    docker exec -i -e PGPASSWORD="$([ "$1" = pgadmin ] && echo bench || echo "bench-$1")" "$name" \
        psql -X -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$1" -d nava < "$2"
}
{
    echo "### host: $(uname -s), $(nproc 2>/dev/null || echo ?) CPUs; docker $(docker version --format '{{.Server.Version}}')"
    docker info --format 'docker VM: {{.NCPU}} CPUs, {{.MemTotal}} bytes'
    echo "### seed (pgadmin)"; psql_as pgadmin "$dir/01_seed.sql"
    echo "### without the 00005 indexes (pgadmin)"; psql_as pgadmin "$dir/02_drop.sql"
    echo "### BEFORE plans (nava_purge)"; psql_as nava_purge "$dir/03_plans.sql"
    echo "### BEFORE plans, warm (nava_purge)"; psql_as nava_purge "$dir/03_plans.sql"
    echo "### BEFORE writes (pgadmin)"; psql_as pgadmin "$dir/04_writes.sql"
    echo "### 00005 over the seeded tables (nava_migrator)"
    { echo '\timing on'; cat db/migrations/00005_purge_indexes.sql; } > "bin/purge-bench-00005.sql"
    psql_as nava_migrator bin/purge-bench-00005.sql
    rm -f bin/purge-bench-00005.sql
    echo "### AFTER plans (nava_purge)"; psql_as nava_purge "$dir/03_plans.sql"
    echo "### AFTER plans, warm (nava_purge)"; psql_as nava_purge "$dir/03_plans.sql"
    echo "### AFTER writes (pgadmin)"; psql_as pgadmin "$dir/04_writes.sql"
    echo "### sizes (pgadmin)"; psql_as pgadmin "$dir/05_sizes.sql"
    echo "### backlog: half of every table expired (pgadmin)"; psql_as pgadmin "$dir/06_backlog.sql"
    echo "### BACKLOG plans (nava_purge)"; psql_as nava_purge "$dir/03_plans.sql"
} > "$out" 2>&1
echo "wrote $out"
