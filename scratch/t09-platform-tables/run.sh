#!/bin/sh
# T09 benchmark: builds the real schema with nava-db (roles, then migrate as
# nava_migrator), seeds scratch data as the superuser, then captures plans as
# the runtime roles. Writes output.txt next to this script. Needs Docker and Go.
#   sh scratch/t09-platform-tables/run.sh
set -eu
here=$(cd "$(dirname "$0")" && pwd)
cd "$here/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="nava-t09-bench-$$"
MSYS_NO_PATHCONV=1; export MSYS_NO_PATHCONV
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT

docker run -d --name "$name" --shm-size=1g -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench \
    -p 127.0.0.1::5432 "postgres:$tag" -c shared_buffers=1GB -c max_wal_size=4GB >/dev/null
until docker exec "$name" pg_isready -h 127.0.0.1 -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
port=$(docker port "$name" 5432/tcp | head -1 | sed 's/.*://')

export PGHOST=127.0.0.1 PGPORT="$port" PGSSLMODE=disable
for r in nava_migrator nava_app nava_worker nava_relay nava_purge nava_onboard nava_groupsync nava_backup keycloak temporal pgbouncer_auth; do
    export "DBROLES_PASSWORD_$(echo "$r" | tr a-z A-Z)=bench-$r"
done
PGUSER=pgadmin PGPASSWORD=bench go run ./cmd/nava-db roles >/dev/null
PGUSER=nava_migrator PGPASSWORD=bench-nava_migrator go run ./cmd/nava-db migrate >/dev/null

out="$here/output.txt"
{
    echo "# T09 benchmark, $(date -u +%Y-%m-%dT%H:%MZ)"
    docker exec "$name" psql -U pgadmin -d nava -Atc 'SELECT version()'
    echo "# host: $(docker info --format '{{.NCPU}} CPUs, {{.MemTotal}} B, {{.OperatingSystem}}'), shared_buffers=1GB"
    echo
    docker exec -i "$name" psql -U pgadmin -d nava -X < "$here/seed.sql"
    docker exec -i -e BIG="$(printf %s tenant1 | md5sum | sed 's/\(........\)\(....\)\(....\)\(....\)\(............\).*/\1-\2-\3-\4-\5/')" \
        -e SMALL="$(printf %s tenant5 | md5sum | sed 's/\(........\)\(....\)\(....\)\(....\)\(............\).*/\1-\2-\3-\4-\5/')" \
        "$name" psql -U pgadmin -d nava -X < "$here/plans.sql"
} > "$out" 2>&1
echo "wrote $out"
