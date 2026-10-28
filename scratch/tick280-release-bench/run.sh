#!/bin/sh
# TICK-280: the plan of the domain release (DELETE FROM tenant_email_domain
# WHERE tenant_id = $1 as nava_onboard, FORCE RLS, email_domain_release and
# email_domain_onboard_select) at 25,003 rows, without and with a (tenant_id)
# index. A throwaway postgres from docs/pins.md, db-roles and every migration
# through nava-db, as docs/data/identity-bench/run.sh. Needs Docker and Go.
#   sh scratch/tick280-release-bench/run.sh
set -eu
cd "$(dirname "$0")/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="nava-tick280-bench-$$"
dir=scratch/tick280-release-bench
out=bin/tick280-release-bench-output.txt
bin=bin/tick280-bench-nava-db$(go env GOEXE)
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
    docker exec "$name" psql -XAt -U pgadmin -d nava -c 'SELECT version()'
    echo "### seed (pgadmin)"; psql_as pgadmin "$dir/01_seed.sql"
    echo "### release (nava_onboard)"; psql_as nava_onboard "$dir/02_release.sql"
    echo "### release, warm (nava_onboard)"; psql_as nava_onboard "$dir/02_release.sql"
    echo "### variant: (tenant_id) index (pgadmin)"; psql_as pgadmin "$dir/03_variant.sql"
    echo "### release with the index (nava_onboard)"; psql_as nava_onboard "$dir/02_release.sql"
    echo "### release with the index, warm (nava_onboard)"; psql_as nava_onboard "$dir/02_release.sql"
} > "$out" 2>&1
echo "wrote $out"
