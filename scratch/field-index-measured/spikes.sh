#!/bin/sh
# TICK-1009 (A1-7 S1, S2) then TICK-1039 (A1-6 arms) in one idle-host window,
# on the TICK-992 harness: run.sh with NAVA_PERF_PHASES=edges,arms. One 800k
# seed serves both; edges runs before arms because arms rewrites the records
# heap (VACUUM FULL after its scratch column). See the header of
# internal/server/records_fieldindex_spikes_perf_integration_test.go.
#
# Host gate (no waiver): no nava, nava-*, nava-it-* or bench containers; the
# report prints the transport floor and the other containers before and after
# each phase. About 60 minutes at the default 20 s per cell.
#   sh scratch/field-index-measured/spikes.sh
# Env: NAVA_PERF_LIST_SECONDS (20); NAVA_PERF_OUT (bin/field-index-spikes-raw.md).
set -eu
cd "$(dirname "$0")/../.."
busy=$(docker ps --format '{{.Names}}' | grep -E '^(nava($|-)|bench)' || true)
if [ -n "$busy" ] && [ "${SPIKES_SMOKE:-0}" != 1 ]; then
    echo "host is not idle: $busy (set SPIKES_SMOKE=1 for an unscored smoke run)" >&2
    exit 1
fi
NAVA_PERF_PHASES=edges,arms
NAVA_PERF_OUT=${NAVA_PERF_OUT:-$(pwd)/bin/field-index-spikes-raw.md}
export NAVA_PERF_PHASES NAVA_PERF_OUT
exec sh scratch/field-index-measured/run.sh
