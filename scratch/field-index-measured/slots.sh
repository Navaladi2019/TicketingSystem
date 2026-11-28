#!/bin/sh
# TICK-1205 (ADR-006-A3 draft, A3-10): promoted filter slots on the TICK-992
# harness: run.sh with NAVA_PERF_PHASES=slots. Write path, A3-2 migration and
# A3-5 backfill under the live mix, sizes, the carry curve, S-b1..S-b3 and
# S-c1..S-c3. See the header of
# internal/server/records_fieldindex_slots_perf_integration_test.go.
#
# Host gate (no waiver): no nava, nava-*, nava-it-* or bench containers; the
# report prints the transport floor and the other containers before and
# after. About 2 hours at the defaults.
#   sh scratch/field-index-measured/slots.sh
# Env: NAVA_PERF_LIST_SECONDS (20), NAVA_PERF_SLOT_WRITE_SECONDS (30),
# NAVA_PERF_SLOT_LIVE_SECONDS (60), NAVA_PERF_SLOT_CURVE (0,8,16,32,64),
# NAVA_PERF_SLOT_COPIES (4); NAVA_PERF_OUT (bin/field-index-slots-raw.md).
set -eu
cd "$(dirname "$0")/../.."
busy=$(docker ps --format '{{.Names}}' | grep -E '^(nava($|-)|bench)' || true)
if [ -n "$busy" ] && [ "${SPIKES_SMOKE:-0}" != 1 ]; then
    echo "host is not idle: $busy (set SPIKES_SMOKE=1 for an unscored smoke run)" >&2
    exit 1
fi
NAVA_PERF_PHASES=slots
NAVA_PERF_OUT=${NAVA_PERF_OUT:-$(pwd)/bin/field-index-slots-raw.md}
export NAVA_PERF_PHASES NAVA_PERF_OUT
exec sh scratch/field-index-measured/run.sh
