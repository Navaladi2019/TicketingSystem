#!/bin/sh
# P3-36 (TICK-621): list p95 <= 150 ms and single read p95 <= 50 ms at 800k
# records, through the real handlers with the compiled ACL SQL. Starts a
# throwaway PostgreSQL with P3-01's memory settings (bench/records-list/run.sh)
# and runs TestRecordsACLPerf800k (internal/server, tags integration perf),
# which seeds P3-01's shape and writes the raw report. Report:
# docs/quality/acl-perf-800k.md. Needs Docker and Go. Not part of CI.
#   make bench-acl-perf        # about 20 minutes
# Env: NAVA_PERF_SECONDS (20) per scenario, NAVA_PERF_CLIENTS (8),
# NAVA_PERF_OUT (bin/acl-perf-800k-raw.md), BENCH_KEEP=1 keeps the container.
set -eu
cd "$(dirname "$0")/../.."
pgtag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$pgtag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
pg=nava-aclperf-pg-$$
NAVA_PERF_OUT=${NAVA_PERF_OUT:-$(pwd)/bin/acl-perf-800k-raw.md}
# go test on Windows reads a Git Bash path (/c/...) as C:\c\...: hand it C:/...
if command -v cygpath >/dev/null 2>&1; then NAVA_PERF_OUT=$(cygpath -m "$NAVA_PERF_OUT"); fi
export NAVA_PERF_OUT
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

cleanup() {
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
    echo "### $(date -u +%Y-%m-%dT%H:%M:%SZ) postgres:$pgtag"
    echo "### host: $(uname -s); docker $(docker version --format '{{.Server.Version}}')"
    docker info --format '### docker VM: {{.NCPU}} CPUs, {{.MemTotal}} bytes, kernel {{.KernelVersion}}'
} | tee "$NAVA_PERF_OUT.host"
go test -tags 'integration perf' -count=1 -timeout 120m -v -run '^TestRecordsACLPerf800k$' ./internal/server
