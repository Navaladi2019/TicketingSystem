#!/bin/sh
# P3-30 record_shares index bench. Needs Docker. Writes result.txt next to it.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
name=acl-tables-bench-$$
docker run -d --rm --name "$name" -e POSTGRES_PASSWORD=bench postgres:16.10 >/dev/null
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
until docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sleep 2
docker exec -i "$name" psql -U postgres -v ON_ERROR_STOP=1 -q < "$here/bench.sql" > "$here/result.txt" 2>&1
