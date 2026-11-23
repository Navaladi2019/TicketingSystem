#!/bin/sh
# SP-1c (TICK-1038): capped-probe region leaf on a kept S31-schema bench.
# Usage: SP1C_PG=<kept container> sh bench/acl-perf/region/probe.sh <setup|explain|paired>
# setup runs region.sql (if needed), closure.sql, then the skewed districts
# and the live-only closure build. Raw output: bin/sp1c-raw.md (appended).
set -eu
cd "$(dirname "$0")/../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
pg=${SP1C_PG:?set SP1C_PG to the kept bench container}
port=$(docker port "$pg" 5432/tcp | head -n 1 | sed 's/.*://')
SP1C_SUPER_URL="postgres://pgadmin:bench@127.0.0.1:$port/nava"
SP1C_APP_URL="postgres://nava_app:test-pw-nava_app@127.0.0.1:$port/nava"
SP1C_OUT=${SP1C_OUT:-$(pwd)/bin/sp1c-raw.md}
if command -v cygpath >/dev/null 2>&1; then SP1C_OUT=$(cygpath -m "$SP1C_OUT"); fi
SP1C_PHASE=$1
export SP1C_PG SP1C_SUPER_URL SP1C_APP_URL SP1C_OUT SP1C_PHASE
if [ "$1" = setup ]; then
    if ! docker exec "$pg" psql -U pgadmin -d nava -Atc "SELECT 1 FROM bench_ou LIMIT 1" >/dev/null 2>&1; then
        docker exec -i "$pg" psql -q -U pgadmin -d nava < bench/acl-perf/region/region.sql
    fi
    docker exec -i "$pg" psql -q -U pgadmin -d nava < bench/acl-perf/region/closure.sql
    docker exec "$pg" psql -q -U pgadmin -d nava -c "DROP INDEX IF EXISTS records_bench_org_unit; DROP INDEX IF EXISTS records_bench_org_unit_updated; CREATE EXTENSION IF NOT EXISTS pg_prewarm"
fi
go run -tags perf ./bench/acl-perf/region/probe
