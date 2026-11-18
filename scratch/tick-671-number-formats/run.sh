#!/bin/sh
# TICK-671: runs bench.sql on a throwaway postgres (tag from docs/pins.md) and
# writes result.txt. Usage: sh run.sh (needs Docker).
set -eu
cd "$(dirname "$0")"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' ../../docs/pins.md)
c=tick671-bench-$$
docker run -d --name "$c" -e POSTGRES_PASSWORD=bench "postgres:$tag" >/dev/null
trap 'docker rm -fv "$c" >/dev/null 2>&1 || true' EXIT
until docker exec "$c" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
{
    echo "postgres:$tag; host $(uname -sm); Docker VM $(docker info --format '{{.NCPU}} CPUs, {{.MemTotal}} bytes')"
    docker exec -i "$c" psql -U postgres -X -v ON_ERROR_STOP=1 < bench.sql
} > result.txt 2>&1
