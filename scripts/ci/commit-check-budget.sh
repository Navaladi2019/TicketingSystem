#!/bin/sh
# The commit-check budget rows (ADR-004-E3 "Budget", D-E3-4, TICK-509):
# TestCommitCheckBudget alone, against one throwaway PostgreSQL cluster from
# the docs/pins.md postgres image, with nothing else running beside it. The
# rows carry absolute µs review thresholds (report-only in CI, ADR-004-E4),
# so the test wants an uncontended runner: in the integration job it shared
# the runner with the dbtest shards and the other groups, and both arms read
# 1.5-7x slow. test-integration.sh leaves it
# out of its shards for that reason; the ci.yml job commit-check-budget runs
# this script on a runner of its own. Needs Docker. Extra arguments go to go
# test. Not part of make ci: a laptop is not the baseline (docs/budgets.md).
#
# TestP511CommitCheckPooledSession runs here too, after the budget test, for
# the same reason (TICK-641): it asserts a timing ratio (median t(4N)/t(N) of
# 3 runs < 8), which is not meaningful on a saturated shared host. In the
# integration job on the loaded self-hosted runner it read 8.05 and failed.
# Its threshold is unchanged; test-integration.sh leaves it out of its shards,
# and this script fails when it did not run.
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
# rm -v: the cluster's anonymous data volume too (TICK-641).
trap 'docker rm -fv "$name" >/dev/null 2>&1 || true' EXIT
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
# The CPU model, so a slow runner host can be told from a regression
# (TICK-509 review finding 3).
cpu=$(sed -n 's/^model name[[:space:]]*: //p' /proc/cpuinfo 2>/dev/null | head -n 1)
# The test puts it on each COMMIT-CHECK-BUDGET line (ADR-004-E4 decision 3).
NAVA_BUDGET_CPU=${cpu:-unknown}
export NAVA_BUDGET_CPU
echo "commit-check-budget ($image): cluster healthy after $(( $(date +%s) - t0 ))s; nproc $(nproc 2>/dev/null || echo ?); cpu=\"${cpu:-unknown}\""

# -timeout well below the job's timeout-minutes, so a hang dumps goroutines.
# The output streams through tee into $out (a FIFO keeps go test's own exit
# status in $gopid), so the row check below can read it.
out=$(mktemp)
fifo=$(mktemp -u)
mkfifo "$fifo"
trap 'docker rm -fv "$name" >/dev/null 2>&1 || true; rm -f "$out" "$fifo"' EXIT
tee "$out" <"$fifo" &
teepid=$!
go test -tags integration -count=1 -timeout 9m -v -run '^(TestCommitCheckBudget|TestP511CommitCheckPooledSession)$' "$@" ./internal/db/dbtest >"$fifo" 2>&1 &
gopid=$!
status=0
wait "$gopid" || status=$?
wait "$teepid" || true
echo "commit-check-budget: done after $(( $(date +%s) - t0 ))s"
# A renamed test makes -run match nothing and go test pass, so a green run
# must have run TestP511CommitCheckPooledSession (TICK-641) and measured both
# budget rows (TICK-509 review finding 2).
if ! grep -q -- '--- PASS: TestP511CommitCheckPooledSession ' "$out" && [ "$status" -eq 0 ]; then
    echo "commit-check-budget: TestP511CommitCheckPooledSession did not run; was it renamed or skipped?" >&2
    status=1
fi
#
# ADR-004-E4: the review thresholds are report-only. A row over its threshold
# in a warn run gets one ::warning:: annotation and the job stays green; the
# test itself fails on fail_no_rows, fail_ratio and fail_flat. Both rows go to
# the step summary when GitHub provides one.
summary=${GITHUB_STEP_SUMMARY:-/dev/null}
{
    echo "### Commit-check budget (ADR-004-E3 method, ADR-004-E4 verdict)"
    echo
    echo '```'
} >>"$summary"
for row in single_subject import; do
    # go test -v indents t.Log lines; keep the line from COMMIT-CHECK-BUDGET on.
    line=$(sed -n "s/.*\(COMMIT-CHECK-BUDGET row=$row .*\)/\1/p" "$out" | head -n 1)
    if [ -z "$line" ]; then
        echo "commit-check-budget: no 'COMMIT-CHECK-BUDGET row=$row' line in the output; was TestCommitCheckBudget renamed or skipped? (fail_no_rows)" >&2
        echo "row=$row: MISSING (fail_no_rows)" >>"$summary"
        status=1
        continue
    fi
    echo "$line" >>"$summary"
    case $line in
        *" over_threshold=true "*"verdict=warn"*)
            echo "::warning title=commit-check budget $row over its review threshold::$line (report-only in CI, ADR-004-E4; compare with the docs/budgets.md baseline for this cpu)" ;;
    esac
done
echo '```' >>"$summary"
exit "$status"
