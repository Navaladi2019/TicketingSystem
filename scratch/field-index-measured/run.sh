#!/bin/sh
# TICK-747 (PLAN-007 P7-07): the S21 field index measured on the merged code
# against ADR-006 D10 and D13 at 800k records. Starts a throwaway PostgreSQL
# with P3-01's memory settings (as bench/acl-perf/run.sh does) and runs
# TestFieldIndexPerf800k (internal/server, tags integration perf): lists
# through the real handler, creates and patches through the real pipeline,
# the real FieldIndexBackfill on a Temporal dev server (the SDK downloads the
# temporal CLI unless NAVA_PERF_TEMPORAL_CLI names one). Report:
# docs/data/field-index-measured.md. Needs Docker, Go and network for the CLI.
# Not part of CI. About 90 minutes at the defaults.
#   sh scratch/field-index-measured/run.sh
# Env: see the test's header (NAVA_PERF_LIST_SECONDS, NAVA_PERF_WRITE_SECONDS,
# NAVA_PERF_WRITE_ROUNDS, NAVA_PERF_LIVE_SECONDS, NAVA_PERF_LIVE_MAX,
# NAVA_PERF_PHASES, NAVA_PERF_CLIENTS, NAVA_PERF_OUT); BENCH_KEEP=1 keeps the
# container.
set -eu
cd "$(dirname "$0")/../.."
pgtag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$pgtag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
pg=nava-fiperf-pg-$$
NAVA_PERF_OUT=${NAVA_PERF_OUT:-$(pwd)/bin/field-index-perf-raw.md}
if command -v cygpath >/dev/null 2>&1; then NAVA_PERF_OUT=$(cygpath -m "$NAVA_PERF_OUT"); fi
export NAVA_PERF_OUT
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

sampler=""
cleanup() {
    [ -n "$sampler" ] && kill "$sampler" 2>/dev/null || true
    if [ "${BENCH_KEEP:-0}" = 1 ]; then echo "kept: $pg"; return; fi
    docker rm -f "$pg" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# P3-01's server: defaults except memory, warm cache, fsync on.
docker run -d --name "$pg" --shm-size=3g -p 127.0.0.1::5432 \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench "postgres:$pgtag" \
    -c shared_buffers=2GB -c effective_cache_size=6GB -c max_wal_size=8GB \
    -c track_io_timing=on >/dev/null
until docker exec "$pg" pg_isready -h 127.0.0.1 -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
port=$(docker port "$pg" 5432/tcp | head -n 1 | sed 's/.*://')
NAVA_TEST_PG_URL="postgres://pgadmin:bench@127.0.0.1:$port/postgres"
export NAVA_TEST_PG_URL
mkdir -p "$(dirname "$NAVA_PERF_OUT")"
{
    echo "### $(date -u +%Y-%m-%dT%H:%M:%SZ) postgres:$pgtag, commit $(git rev-parse --short HEAD)"
    echo "### host: $(uname -s); docker $(docker version --format '{{.Server.Version}}')"
    docker info --format '### docker VM: {{.NCPU}} CPUs, {{.MemTotal}} bytes, kernel {{.KernelVersion}}'
    echo "### other containers at start: $(docker ps --format '{{.Names}}' | grep -v "^$pg\$" | tr '\n' ' ')"
} | tee "$NAVA_PERF_OUT.host"
# Every 10 s, the other containers on the Docker VM and their CPU: a phase
# that ran beside a busy one is contended.
(
    while :; do
        echo "$(date -u +%H:%M:%S) $(docker stats --no-stream --format '{{.Name}}={{.CPUPerc}}' | grep -v "^$pg=" | tr '\n' ' ')"
        sleep 10
    done
) > "$NAVA_PERF_OUT.contention" 2>/dev/null &
sampler=$!
go test -tags 'integration perf' -count=1 -timeout 240m -v -run '^TestFieldIndexPerf800k$' ./internal/server
