#!/bin/sh
# Runs bench.sql in a throwaway container of the docs/pins.md postgres pin.
set -eu
cd "$(dirname "$0")"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' ../../docs/pins.md)
name=nava-ik-headers-bench-$$
docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=x "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null' EXIT
until docker exec "$name" pg_isready -U postgres -q; do sleep 1; done
sleep 2
docker exec "$name" psql -U postgres -tAc 'select version()'
docker exec -i "$name" psql -U postgres -v ON_ERROR_STOP=1 < bench.sql
