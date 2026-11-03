#!/bin/sh
# The commit-check budget rows (ADR-004-E3 "Budget", D-E3-4, TICK-509):
# TestCommitCheckBudget alone, against one throwaway PostgreSQL cluster from
# the docs/pins.md postgres image, with nothing else running beside it. The
# rows are absolute µs ceilings, so the test needs an uncontended runner: in
# the integration job it shared the runner with the dbtest shards and the
# other groups, and both arms read 1.5-7x slow. test-integration.sh leaves it
# out of its shards for that reason; the ci.yml job commit-check-budget runs
# this script on a runner of its own. Needs Docker. Extra arguments go to go
# test. Not part of make ci: a laptop is not the baseline (docs/budgets.md).
set -eu
cd "$(dirname "$0")/../.."

tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
image="postgres:$tag"
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

# Named and labelled like test-integration.sh's clusters (nava-it-*), so make
# reap-integration removes what a killed run left.
id=${NAVA_IT_ID:-$$$(od -An -N2 -tu2 /dev/urandom | awk '{printf "%05d", $1}')}
case $id in '' | *[!0-9]*) echo "NAVA_IT_ID=$id: digits only" >&2; exit 2 ;; esac
owner=${COMPOSE_PROJECT_NAME:-none}
name="nava-it-pg-budget-$id"
gopid=""
trap 'docker rm -f "$name" >/dev/null 2>&1 || true' EXIT
trap '[ -z "$gopid" ] || kill "$gopid" 2>/dev/null || true; exit 130' INT
trap '[ -z "$gopid" ] || kill "$gopid" 2>/dev/null || true; exit 143' TERM

t0=$(date +%s)
# The integration clusters' flags (no durability), so the numbers compare
# with the TA's S2 runs, which used the same.
docker run -d --name "$name" --label "nava.ci.run=nava-it-$id" --label "nava.ci.owner=$owner" \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only -e POSTGRES_INITDB_ARGS=--no-sync \
    --health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
    --health-interval 1s --health-timeout 5s --health-retries 90 \
    -p 127.0.0.1::5432 "$image" \
    -c fsync=off -c synchronous_commit=off -c full_page_writes=off >/dev/null
# Compile while the cluster starts; -run '^$' runs nothing.
go test -tags integration -count=1 -run '^$' ./internal/db/dbtest >/dev/null

deadline=$(( $(date +%s) + 120 ))
while :; do
    st=$(docker inspect -f '{{.State.Status}} {{.State.Health.Status}}' "$name")
    case $st in
        "running healthy") break ;;
        "running unhealthy" | exited* | dead*) docker logs "$name" 2>&1 | tail -n 50; echo "$name is $st" >&2; exit 1 ;;
    esac
    [ "$(date +%s)" -lt "$deadline" ] || { docker logs "$name"; echo "$name not healthy within 120s" >&2; exit 1; }
    sleep 1
done
NAVA_TEST_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$(docker port "$name" 5432/tcp | head -n 1 | sed 's/.*://')/postgres"
export NAVA_TEST_PG_URL
echo "commit-check-budget ($image): cluster healthy after $(( $(date +%s) - t0 ))s; nproc $(nproc 2>/dev/null || echo ?)"

# -timeout well below the job's timeout-minutes, so a hang dumps goroutines.
go test -tags integration -count=1 -timeout 6m -v -run '^TestCommitCheckBudget$' "$@" ./internal/db/dbtest &
gopid=$!
status=0
wait "$gopid" || status=$?
echo "commit-check-budget: done after $(( $(date +%s) - t0 ))s"
exit "$status"
