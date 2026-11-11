#!/bin/sh
# P3-40 (TICK-709): runs bench.sql on a throwaway postgres container (the
# docs/pins.md tag) and removes it. Output: the EXPLAIN (ANALYZE, BUFFERS)
# plans and sizes in docs/data/workflow-definitions.md.
set -eu
cd "$(dirname "$0")/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="p340-bench-$$"
docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=bench "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec -i "$name" psql -X -q -v ON_ERROR_STOP=1 -U postgres < scratch/p340-workflow-definitions/bench.sql
