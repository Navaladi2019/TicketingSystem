#!/bin/sh
# TICK-1032 records_org_unit swap measurement: docs/data/org-unit-closure.md. A throwaway postgres container
# with the docs/pins.md tag; raw output in swap-out.txt next to this file.
# About 1 minute.
set -eu
cd "$(dirname "$0")"
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' ../../docs/pins.md)
name=nava-tick1032-swap-$$
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
docker run -d --rm --shm-size=1g --name "$name" -e POSTGRES_PASSWORD=bench -e POSTGRES_DB=bench "postgres:$tag" \
    -c shared_buffers=1GB -c work_mem=16MB -c maintenance_work_mem=512MB >/dev/null
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
until docker exec "$name" pg_isready -U postgres -d bench >/dev/null 2>&1; do sleep 1; done
sleep 2
docker cp swap.sql "$name":/tmp/swap.sql
{
    echo "host: $(uname -a)"
    docker info --format 'docker VM: {{.NCPU}} CPUs, {{.MemTotal}} bytes'
    echo "containers running: $(docker ps --format '{{.Names}}' | tr '\n' ' ')"
    docker exec "$name" psql -X -U postgres -d bench -f /tmp/swap.sql
} > swap-out.txt 2>&1
echo "wrote $(pwd)/swap-out.txt"
