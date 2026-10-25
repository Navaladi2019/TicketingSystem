#!/bin/sh
# T16 (TICK-69) identity-table benchmark: a throwaway postgres from
# docs/pins.md, db-roles and every migration through nava-db, the seed in
# 01_seed.sql, then the plans as the runtime roles with RLS on. Output in
# bin/identity-bench-output.txt (gitignored). Needs Docker and Go.
#   sh docs/data/identity-bench/run.sh
set -eu
cd "$(dirname "$0")/../../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="nava-t16-bench-$$"
dir=docs/data/identity-bench
out=bin/identity-bench-output.txt
bin=bin/identity-bench-nava-db$(go env GOEXE)
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -f "$bin"' EXIT

mkdir -p bin
go build -o "$bin" ./cmd/nava-db
docker run -d --name "$name" --shm-size=1g -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench \
    -p 127.0.0.1::5432 "postgres:$tag" -c shared_buffers=1GB >/dev/null
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
    echo "### resolver (nava_app)"; psql_as nava_app "$dir/02_resolver.sql"
    echo "### resolver, second run (warm)"; psql_as nava_app "$dir/02_resolver.sql"
    echo "### sync (nava_groupsync)"; psql_as nava_groupsync "$dir/03_sync.sql"
    echo "### sizes (pgadmin)"; psql_as pgadmin "$dir/04_sizes.sql"
    echo "### variant: three-column membership index (pgadmin)"; psql_as pgadmin "$dir/05_variant.sql"
    echo "### resolver with the three-column index (nava_app)"; psql_as nava_app "$dir/02_resolver.sql"
    echo "### resolver with the three-column index, warm (nava_app)"; psql_as nava_app "$dir/02_resolver.sql"
} > "$out" 2>&1
echo "wrote $out"
