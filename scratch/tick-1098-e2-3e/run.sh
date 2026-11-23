#!/bin/sh
# TICK-1098 (PLAN-007-E2 T-E2-2): ADR-006-E2 E2-4 (3e), the serial-count
# check, and the reworded (3a) leak reading (E2-3), on T-E2-1's code. Same
# shape as scratch/tick-1048-a2-gate/run.sh: a fresh throwaway PostgreSQL with
# P3-01's memory settings, then TestA2Gate (internal/server, tags integration
# perf) with NAVA_PERF_SPIKE_CELLS=serial and NAVA_PERF_GATE_ATTEMPTS=0 (the
# (2) budget check is not rerun). Needs Docker and Go. Not part of CI. About
# 15 min on a quiet host (estimate). Report:
# docs/data/adr-006-a2-gate/run-e2-3e/ (raw.md, plans.md, host.txt,
# contention.txt, serial-samples.csv). Env: NAVA_PERF_LIST_SECONDS (15),
# NAVA_PERF_CLIENTS (8), NAVA_PERF_OUT; BENCH_KEEP=1 keeps the container.
#   sh scratch/tick-1098-e2-3e/run.sh
set -eu
cd "$(dirname "$0")/../.."
pgtag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$pgtag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
pg=nava-e2-3e-pg-$$
NAVA_PERF_OUT=${NAVA_PERF_OUT:-$(pwd)/docs/data/adr-006-a2-gate/run-e2-3e/raw.md}
if command -v cygpath >/dev/null 2>&1; then NAVA_PERF_OUT=$(cygpath -m "$NAVA_PERF_OUT"); fi
out=$(dirname "$NAVA_PERF_OUT")
NAVA_PERF_SPIKE_CELLS=${NAVA_PERF_SPIKE_CELLS:-serial}
NAVA_PERF_GATE_ATTEMPTS=${NAVA_PERF_GATE_ATTEMPTS:-0}
export NAVA_PERF_OUT NAVA_PERF_SPIKE_CELLS NAVA_PERF_GATE_ATTEMPTS
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

sampler=""
cleanup() {
    [ -n "$sampler" ] && kill "$sampler" 2>/dev/null || true
    if [ "${BENCH_KEEP:-0}" = 1 ]; then echo "kept: $pg"; return; fi
    docker rm -fv "$pg" >/dev/null 2>&1 || true # -v: the data volume too
}
trap cleanup EXIT

# P3-01's server: defaults except memory, warm cache, fsync on. Parallel
# settings stay at the server defaults (arm B is "server settings").
docker run -d --name "$pg" --shm-size=3g -p 127.0.0.1::5432 \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=bench "postgres:$pgtag" \
    -c shared_buffers=2GB -c effective_cache_size=6GB -c max_wal_size=8GB \
    -c track_io_timing=on >/dev/null
until docker exec "$pg" pg_isready -h 127.0.0.1 -U pgadmin -d postgres >/dev/null 2>&1; do sleep 1; done
port=$(docker port "$pg" 5432/tcp | head -n 1 | sed 's/.*://')
NAVA_PERF_PG_CONTAINER=$pg
export NAVA_PERF_PG_CONTAINER
NAVA_TEST_PG_URL="postgres://pgadmin:bench@127.0.0.1:$port/postgres"
export NAVA_TEST_PG_URL
mkdir -p "$out"
{
    echo "### $(date -u +%Y-%m-%dT%H:%M:%SZ) postgres:$pgtag, commit $(git rev-parse --short HEAD)"
    echo "### host: $(uname -s); docker $(docker version --format '{{.Server.Version}}')"
    docker info --format '### docker VM: {{.NCPU}} CPUs, {{.MemTotal}} bytes, kernel {{.KernelVersion}}'
    echo "### other containers at start: $(docker ps --format '{{.Names}}' | grep -v "^$pg\$" | tr '\n' ' ')"
    echo "### NAVA_PERF_SPIKE_CELLS=$NAVA_PERF_SPIKE_CELLS NAVA_PERF_GATE_ATTEMPTS=$NAVA_PERF_GATE_ATTEMPTS"
} | tee "$out/host.txt"
# Every 10 s, the Docker VM's load average, the other containers' CPU and,
# on a Windows host, the count of host-side Go, lint and test processes
# outside Docker (TICK-1098 window 1: other sessions' go and golangci-lint
# ran beside the gates, which docker stats cannot see). server.test is this
# run's own binary.
hostprocs() {
    command -v tasklist >/dev/null 2>&1 || return 0
    tasklist 2>/dev/null | awk '{print $1}' | grep -i -E '^(golangci-lint|go|.*[.]test|node)[.]exe$' |
        sort | uniq -c | awk '{printf "%s=%s ", $2, $1}'
}
(
    while :; do
        echo "$(date -u +%H:%M:%S) load $(docker exec "$pg" cut -d' ' -f1-3 /proc/loadavg 2>/dev/null) $(docker stats --no-stream --format '{{.Name}}={{.CPUPerc}}' | grep -v "^$pg=" | tr '\n' ' ') host: $(hostprocs)"
        sleep 10
    done
) > "$out/contention.txt" 2>/dev/null &
sampler=$!
go test -tags 'integration perf' -count=1 -timeout 120m -v -run '^TestA2Gate$' ./internal/server
