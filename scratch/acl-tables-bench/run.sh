#!/bin/sh
# P3-30 record_shares index bench. Needs Docker. Writes result.txt next to it.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
# The image tag comes from the postgres row of docs/pins.md, as the CI scripts read it.
tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' "$here/../../docs/pins.md")
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
name=acl-tables-bench-$$
docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=bench "postgres:$tag" >/dev/null
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec -i "$name" psql -U postgres -v ON_ERROR_STOP=1 -q < "$here/bench.sql" > "$here/result.txt" 2>&1
