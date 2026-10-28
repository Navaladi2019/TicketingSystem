#!/bin/sh
# TICK-250: EXPLAIN (ANALYZE, BUFFERS) of claimSQL as pgx runs it: a prepared
# statement, custom plan then forced generic plan, rolled back.
#   PSQL="docker exec -i pg psql" sh scratch/tick-250/generic.sh <psql args...>
# REV picks the relay.go revision (default: working tree).
set -eu
cd "$(dirname "$0")/../.."
src() { if [ -n "${REV:-}" ]; then git show "$REV:internal/relay/relay.go"; else cat internal/relay/relay.go; fi; }
sql=$(src | sed -n '/^const claimSQL = `/,/`$/p' | sed '1s/^const claimSQL = `//; $s/`$//')
for mode in force_custom_plan force_generic_plan; do
    echo "=== $mode"
    { echo 'BEGIN;'; echo "SET LOCAL plan_cache_mode = $mode;"
      echo "PREPARE claim(int, int, float8, float8, text) AS $sql;"
      echo "EXPLAIN (ANALYZE, BUFFERS) EXECUTE claim(200, 2, 30, 86400, 'expired');"
      echo 'ROLLBACK;'; } | ${PSQL:-psql} -X -q "$@"
done
