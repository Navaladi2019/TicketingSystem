#!/bin/sh
# T22a (TICK-212) idempotency-lease benchmark: a throwaway postgres from
# docs/pins.md, db-roles and every migration through nava-db, then 00006 is
# undone (DROP COLUMN, catalogue only), 01_seed.sql loads 500,000 rows, and
# 00006 is applied again over them, timed, as nava_migrator. Then the takeover
# plans as nava_app with RLS on, the write cost per takeover before (T22's
# FOR UPDATE) and after (the lease UPDATE), and the storage cost.
# full_page_writes is off so 04_writes.sql measures records only.
# Output in bin/lease-bench-output.txt (gitignored). Needs Docker and Go.
#   sh docs/data/lease-bench/run.sh
set -eu
cd "$(dirname "$0")/../../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="nava-t212-bench-$$"
dir=docs/data/lease-bench
out=bin/lease-bench-output.txt
bin=bin/lease-bench-nava-db$(go env GOEXE)
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
trap 'docker rm -f "$name" >/dev/null 2>&1 || true; rm -f "$bin" bin/lease-bench-00006.sql' EXIT

mkdir -p bin
go build -o "$bin" ./cmd/nava-db
docker run -d --name "$name" --shm-size=1g -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench \
    -p 127.0.0.1::5432 "postgres:$tag" -c shared_buffers=1GB -c full_page_writes=off >/dev/null
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
    echo "### undo 00006 (nava_migrator)"
    echo 'ALTER TABLE idempotency_keys DROP COLUMN lease_until;' > bin/lease-bench-00006.sql
    psql_as nava_migrator bin/lease-bench-00006.sql
    echo "### seed (pgadmin)"; psql_as pgadmin "$dir/01_seed.sql"
    echo "### 00006 over the seeded table (nava_migrator)"
    { echo '\timing on'; echo 'BEGIN;'; cat db/migrations/00006_idempotency_lease.sql; echo 'COMMIT;'; } > bin/lease-bench-00006.sql
    psql_as nava_migrator bin/lease-bench-00006.sql
    # 00006 is recorded as applied, so this only re-applies grants.yaml to the new column.
    PGHOST=127.0.0.1 PGPORT=$port PGUSER=nava_migrator PGPASSWORD=bench-nava_migrator PGDATABASE=nava "$bin" migrate
    echo "### heap after 00006 (pgadmin)"
    echo "SELECT pg_relation_size('idempotency_keys') AS heap_bytes, pg_relation_filenode('idempotency_keys') AS filenode;" > bin/lease-bench-00006.sql
    psql_as pgadmin bin/lease-bench-00006.sql
    echo "### leases as written by the middleware (pgadmin)"; psql_as pgadmin "$dir/02_lease.sql"
    echo "### sizes (pgadmin)"; psql_as pgadmin "$dir/05_sizes.sql"
    echo "### plans (nava_app)"; psql_as nava_app "$dir/03_plans.sql"
    echo "### plans, warm (nava_app)"; psql_as nava_app "$dir/03_plans.sql"
    echo "### writes (pgadmin)"; psql_as pgadmin "$dir/04_writes.sql"
} > "$out" 2>&1
echo "wrote $out"
