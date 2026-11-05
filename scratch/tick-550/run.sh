#!/bin/sh
# TICK-550: run repro.go through each PgBouncer tag and pgx exec mode.
#   sh scratch/tick-550/run.sh [pgbouncer-tag ...]
# Postgres image and default PgBouncer tag come from docs/pins.md.
set -eu
cd "$(dirname "$0")/../.."
pg_tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ $# -gt 0 ] || set -- "$(sed -n 's/^| `edoburu\/pgbouncer` | \([^ |]*\) |.*/\1/p' docs/pins.md)"
net=tick550; docker network create $net >/dev/null 2>&1 || true
docker rm -f tick550-pg >/dev/null 2>&1 || true
docker run -d --rm --name tick550-pg --network $net -p 127.0.0.1:55433:5432 -e POSTGRES_PASSWORD=pw \
    -e POSTGRES_HOST_AUTH_METHOD=scram-sha-256 -e POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256 \
    "postgres:$pg_tag" >/dev/null
until docker exec tick550-pg pg_isready -U postgres -h 127.0.0.1 >/dev/null 2>&1; do sleep 1; done
sleep 1
for tag in "$@"; do
    docker rm -f tick550-bouncer >/dev/null 2>&1 || true
    docker run -d --rm --name tick550-bouncer --network $net -p 127.0.0.1:55432:5432 \
        -e DB_HOST=tick550-pg -e DB_USER=postgres -e DB_PASSWORD=pw -e AUTH_TYPE=scram-sha-256 \
        -e POOL_MODE=transaction -e MAX_PREPARED_STATEMENTS=100 -e DEFAULT_POOL_SIZE=1 \
        "edoburu/pgbouncer:$tag" >/dev/null
    sleep 2
    for mode in cache-statement cache-describe exec; do
        echo "### pgbouncer $tag, pgx $mode"
        go run scratch/tick-550/repro.go "postgres://postgres:pw@127.0.0.1:55432/postgres?sslmode=disable" $mode || true
    done
    docker logs tick550-bouncer 2>&1 | grep -i "prepared\|closing" | tail -4 || true
done
echo "### direct to postgres $pg_tag, pgx cache-statement"
go run scratch/tick-550/repro.go "postgres://postgres:pw@127.0.0.1:55433/postgres?sslmode=disable" cache-statement || true
docker rm -f tick550-bouncer tick550-pg >/dev/null 2>&1 || true
docker network rm $net >/dev/null 2>&1 || true
