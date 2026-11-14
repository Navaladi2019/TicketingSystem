#!/bin/sh
# TICK-785: runs bench.sql on a throwaway postgres container (the docs/pins.md
# tag) and removes it. Output: out.txt, quoted in docs/data/workflow-definitions.md.
set -eu
cd "$(dirname "$0")/../.."
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
name="tick785-bench-$$"
docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=bench "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec -i "$name" psql -X -q -v ON_ERROR_STOP=1 -U postgres < scratch/tick785-workflow-cause-newest/bench.sql
