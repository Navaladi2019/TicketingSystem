#!/bin/sh
# TICK-1048 (PLAN-007-A2 T-A2-4): ADR-006-A2 A2-8 (2), (2a) and (3a) on the
# T-A2-2 code. Starts a fresh throwaway PostgreSQL with P3-01's memory
# settings (as scratch/tick-1022-a2-spikes/run.sh and bench/acl-perf/run.sh
# do) and runs TestA2Gate (internal/server, tags integration perf): the
# server in-process, so the test-only WithSkipCount hook works. Report:
# docs/data/adr-006-a2-gate.md. Needs Docker and Go. Not part of CI. About
# 30 minutes. Env: NAVA_PERF_LIST_SECONDS (15), NAVA_PERF_CLIENTS (8),
# NAVA_PERF_GATE_ATTEMPTS (2), NAVA_PERF_SPIKE_PER_CLIENT (25),
# NAVA_PERF_OUT; BENCH_KEEP=1 keeps the container.
#   sh scratch/tick-1048-a2-gate/run.sh
set -eu
cd "$(dirname "$0")/../.."
pgtag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$pgtag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
pg=nava-a2gate-pg-$$
NAVA_PERF_OUT=${NAVA_PERF_OUT:-$(pwd)/bin/tick-1048-a2-gate/raw.md}
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
go test -tags 'integration perf' -count=1 -timeout 120m -v -run '^TestA2Gate$' ./internal/server
