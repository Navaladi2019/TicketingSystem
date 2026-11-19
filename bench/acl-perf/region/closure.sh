#!/bin/sh
# SP-1b (TICK-969): closure-table region leaf on SP-1's kept bench container.
# Usage: SP1_PG=<kept container> sh bench/acl-perf/region/closure.sh <step>
# Steps: setup (region.sql if needed, closure.sql, build), then
#   plain8a plain8b plain1a plain1b r18a r18b r11a r11b single
# ("a" = area and district ancestors, "b" = region, root and SP-1's D7 |S|=1,000;
# each step measures its own baseline first). Raw output:
# bin/acl-perf-region-closure-raw.md (appended).
set -eu
cd "$(dirname "$0")/../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
pg=${SP1_PG:?set SP1_PG to the kept bench container}
port=$(docker port "$pg" 5432/tcp | head -n 1 | sed 's/.*://')
SP1B_PG=$pg
SP1B_SUPER_URL="postgres://pgadmin:bench@127.0.0.1:$port/nava"
SP1B_APP_URL="postgres://nava_app:test-pw-nava_app@127.0.0.1:$port/nava"
SP1B_OUT=${SP1B_OUT:-$(pwd)/bin/acl-perf-region-closure-raw.md}
if command -v cygpath >/dev/null 2>&1; then SP1B_OUT=$(cygpath -m "$SP1B_OUT"); fi
export SP1B_PG SP1B_SUPER_URL SP1B_APP_URL SP1B_OUT
mkdir -p bin
run() { go run -tags perf ./bench/acl-perf/region/closure; }
case "$1" in
setup)
    if ! docker exec "$pg" psql -U pgadmin -d nava -Atc "SELECT 1 FROM bench_ou LIMIT 1" >/dev/null 2>&1; then
        docker exec -i "$pg" psql -q -U pgadmin -d nava < bench/acl-perf/region/region.sql
    fi
    docker exec -i "$pg" psql -q -U pgadmin -d nava < bench/acl-perf/region/closure.sql
    SP1B_PHASE=build run ;;
plain8a) SP1B_PHASE=plain SP1B_CLIENTS=8 SP1B_SECONDS=20 SP1B_ONLY='area|district' run ;;
plain8b) SP1B_PREP=0 SP1B_PHASE=plain SP1B_CLIENTS=8 SP1B_SECONDS=20 SP1B_ONLY='region|root|D7' run ;;
plain1a) SP1B_PREP=0 SP1B_PHASE=plain SP1B_CLIENTS=1 SP1B_SECONDS=10 SP1B_EXPLAIN=0 SP1B_ONLY='area|district' run ;;
plain1b) SP1B_PREP=0 SP1B_PHASE=plain SP1B_CLIENTS=1 SP1B_SECONDS=10 SP1B_EXPLAIN=0 SP1B_ONLY='region|root|D7' run ;;
r18a) SP1B_PHASE=r1 SP1B_CLIENTS=8 SP1B_SECONDS=20 SP1B_ONLY='area|district' run ;;
r18b) SP1B_PREP=0 SP1B_PHASE=r1 SP1B_CLIENTS=8 SP1B_SECONDS=20 SP1B_ONLY='region|root|D7' run ;;
r11a) SP1B_PREP=0 SP1B_PHASE=r1 SP1B_CLIENTS=1 SP1B_SECONDS=10 SP1B_EXPLAIN=0 SP1B_ONLY='area|district' run ;;
r11b) SP1B_PREP=0 SP1B_PHASE=r1 SP1B_CLIENTS=1 SP1B_SECONDS=10 SP1B_EXPLAIN=0 SP1B_ONLY='region|root|D7' run ;;
single) SP1B_PHASE=single SP1B_CLIENTS=1 SP1B_SECONDS=10 run ;;
*) echo "unknown step $1" >&2; exit 2 ;;
esac
