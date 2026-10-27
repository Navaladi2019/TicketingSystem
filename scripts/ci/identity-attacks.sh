#!/bin/sh
# make identity-attacks (T19, TICK-75): the CI-5 adversarial identity suite
# (internal/identitystack, TestCI5) against the running stack's Keycloak, with
# a throwaway PostgreSQL beside it for the platform database the resolver and
# group sync use. Writes CI5_REPORT (default ci5-report.md), the CI artifact,
# through tools/ci/ci5report, and fails when a CI-5 case did not pass.
#
# make identity-attacks-fixture (`fixture`): the CI-5.06 negative fixture
# against a throwaway Keycloak 26.1.2, where the kc.org tamper must succeed.
# Needs no stack.
#
# Needs Docker. Extra arguments go to `go test`.
set -eu
cd "$(dirname "$0")/../.."
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

mode=suite
if [ "${1:-}" = fixture ]; then
    mode=fixture
    shift
fi

wait_healthy() {
    deadline=$(( $(date +%s) + $2 ))
    until [ "$(docker inspect -f '{{.State.Health.Status}}' "$1")" = healthy ]; do
        [ "$(date +%s)" -lt "$deadline" ] || { docker logs "$1"; echo "$1 not healthy within $2s" >&2; exit 1; }
        sleep 1
    done
}
hostport() { docker port "$1" "$2/tcp" | head -n 1 | sed 's/.*://'; }

if [ "$mode" = fixture ]; then
    # The known-broken version docs/pins.md records as the negative fixture.
    kc_tag=$(sed -n 's/^| `quay.io\/keycloak\/keycloak` | [^|]*(negative fixture: \([0-9.]*\)).*/\1/p' docs/pins.md)
    [ -n "$kc_tag" ] || { echo "no Keycloak negative fixture in docs/pins.md" >&2; exit 1; }
    kc="nava-ci5-kc-fixture-$$"
    trap 'docker rm -f "$kc" >/dev/null 2>&1 || true' EXIT
    trap 'exit 130' INT TERM
    # start-dev, throwaway admin.
    docker run -d --name "$kc" -p 127.0.0.1::8080 \
        -e KC_BOOTSTRAP_ADMIN_USERNAME=admin -e KC_BOOTSTRAP_ADMIN_PASSWORD=dev-only-admin -e KC_HEALTH_ENABLED=true \
        --health-cmd "exec 3<>/dev/tcp/127.0.0.1/9000 && printf 'GET /health/ready HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n' >&3 && head -n1 <&3 | grep -q ' 200 '" \
        --health-interval 3s --health-timeout 5s --health-retries 100 \
        "quay.io/keycloak/keycloak:$kc_tag" start-dev >/dev/null
    go vet -tags identitystack,identityfixture ./internal/identitystack
    wait_healthy "$kc" 300
    NAVA_CI5_KC_FIXTURE_URL="http://127.0.0.1:$(hostport "$kc" 8080)"
    export NAVA_CI5_KC_FIXTURE_URL
    exec go test -tags identitystack,identityfixture -count=1 -v -run '^TestKC2612FixtureOrgPickerTamperSucceeds$' ./internal/identitystack "$@"
fi

tag=$(sed -n 's/^| `postgres` | \([^ |]*\) |.*/\1/p' docs/pins.md)
[ -n "$tag" ] || { echo "no postgres row in docs/pins.md" >&2; exit 1; }
pg="nava-ci5-pg-$$"
trap 'docker rm -f "$pg" >/dev/null 2>&1 || true' EXIT
trap 'exit 130' INT TERM
# Throwaway cluster, so no durability (as isolation.sh).
docker run -d --name "$pg" \
    -e POSTGRES_USER=pgadmin -e POSTGRES_PASSWORD=test-only -e POSTGRES_INITDB_ARGS=--no-sync \
    --health-cmd 'pg_isready -h 127.0.0.1 -U pgadmin -d postgres' \
    --health-interval 1s --health-timeout 5s --health-retries 90 \
    -p 127.0.0.1::5432 "postgres:$tag" \
    -c fsync=off -c synchronous_commit=off -c full_page_writes=off >/dev/null
go vet -tags identitystack ./internal/identitystack
wait_healthy "$pg" 120
NAVA_TEST_PG_URL="postgres://pgadmin:test-only@127.0.0.1:$(hostport "$pg" 5432)/postgres"
export NAVA_TEST_PG_URL

# The CI-5 stream, then the check. Two streams cover the identitystack
# registered suites: this one (TestCI5, ci5_isolation_test.go) and the stack
# one make stack-test-identity writes (every other test in the package, with
# -skip '^TestCI5$'). suiteran -in checks them together: a skip is never
# counted as a run (TICK-230), and the pipe cannot hide a failing go test.
ci5_stream="${CI5_STREAM:-ci5.json}"
stack_stream="${IDENTITY_STREAM:-identity-stream.json}"
status=0
go test -tags identitystack -count=1 -json -run '^TestCI5$' ./internal/identitystack "$@" >"$ci5_stream" || status=1
go run ./tools/ci/ci5report -out "${CI5_REPORT:-ci5-report.md}" <"$ci5_stream" || status=1
# Standalone run: make stack-test-identity has not written its stream.
if [ ! -s "$stack_stream" ]; then
    go test -tags identitystack -count=1 -json -skip '^TestCI5$' ./internal/identitystack >"$stack_stream" || status=1
fi
go run ./tools/ci/suiteran -tags identitystack -in "$stack_stream" "$ci5_stream" || status=1
exit "$status"
