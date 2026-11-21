#!/bin/sh
# SP-1 (TICK-932, PLAN-011 P11-01): region-access cost of
# `org_unit_id IS NOT NULL AND org_unit_id = ANY($S)` on the P3-36 800k seed.
# Report: docs/quality/acl-perf-region.md. Needs Docker and Go. Not in CI.
#
# 1. Seeds the 800k bench with the P3-36 harness and keeps its container; that
#    run also measures the handler baseline for "agent list, sort=-updated_at".
# 2. region.sql adds the org tree and records.org_unit_id in that bench
#    database only (no migration).
# 3. main.go measures the list's statements with and without the region branch,
#    for |S| = 10, 100, 1,000, 10,000, with the plain index and then with
#    R1's fallback index added.
# Env: SP1_SECONDS (20), SP1_CLIENTS (8), SP1_OUT (bin/acl-perf-region-raw.md),
# SP1_INDEX=real (TICK-951: S31's records_org_unit and authz.ExpandUnder).
# Reuse a kept container with SP1_PG=<name> to skip step 1.
set -eu
cd "$(dirname "$0")/../../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV
mkdir -p bin
if [ -z "${SP1_PG:-}" ]; then
    BENCH_KEEP=1 NAVA_PERF_ONLY='^agent list, sort=-updated_at$' \
        NAVA_PERF_OUT="$(pwd)/bin/acl-perf-region-baseline-raw.md" \
        sh bench/acl-perf/run.sh | tee bin/acl-perf-region-seed.log
    SP1_PG=$(sed -n 's/^kept: //p' bin/acl-perf-region-seed.log | tail -n 1)
fi
[ -n "$SP1_PG" ] || { echo "no kept bench container" >&2; exit 1; }
port=$(docker port "$SP1_PG" 5432/tcp | head -n 1 | sed 's/.*://')
SP1_OUT=${SP1_OUT:-$(pwd)/bin/acl-perf-region-raw.md}
# go on Windows reads a Git Bash path (/c/...) as C:\c\...: hand it C:/...
if command -v cygpath >/dev/null 2>&1; then SP1_OUT=$(cygpath -m "$SP1_OUT"); fi
docker exec -i "$SP1_PG" psql -q -U pgadmin -d nava < bench/acl-perf/region/region.sql
SP1_SUPER_URL="postgres://pgadmin:bench@127.0.0.1:$port/nava" \
SP1_APP_URL="postgres://nava_app:test-pw-nava_app@127.0.0.1:$port/nava" \
SP1_OUT="${SP1_OUT:-$(pwd)/bin/acl-perf-region-raw.md}" \
    go run -tags perf ./bench/acl-perf/region
echo "container kept: $SP1_PG (docker rm -f $SP1_PG)"
