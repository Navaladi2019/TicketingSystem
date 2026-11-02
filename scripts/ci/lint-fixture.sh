#!/bin/sh
# The lint rules must report exactly their findings on the fixtures in
# internal/platform/lintfixture: a partly flagged or non-compiling fixture
# fails.
#   forbidigo: 5 raw HTTP client uses (T04, Security Reviewer, TICK-55),
#              1 otelpgx.WithIncludeQueryParameters (T07, TICK-94 item 4) and
#              1 workflow.WithDataConverter (TICK-245).
#   depguard: 1 JWT library import outside internal/identity/keycloak (T15,
#              CI-11).
#   navalint setconfig: 2 tenant settings outside internal/tenancy (T07),
#              seen by both of navalint.sh's passes, plus 1 in an
#              integration-tagged and 1 in a temporalstack-tagged file, seen
#              only by the tagged pass `make lint` runs.
#   navalint idpclaims: 1 Keycloak claim read and 1 admin API path outside
#              internal/identity/keycloak (T15, CI-11), seen by both passes.
#   navalint guarderrtype: 1 reference to workerguard.ErrTypeTenantGuard and
#              1 "nava.WorkflowRefNotFound" literal outside
#              internal/workerguard (AR-044, P3-26), seen by both passes.
#   workflowcheck: 1 range over a map in a workflow function (D-3-23).
set -u
cd "$(dirname "$0")/../.."
want=7
code=0; out=$(${GOLANGCI_LINT:-golangci-lint} run --build-tags lintfixture ./internal/platform/lintfixture/ 2>&1) || code=$?
echo "$out"
test "$code" -eq 1 || { echo "lint-fixture: expected exit 1, got $code"; exit 1; }
n=$(echo "$out" | grep -c "(forbidigo)$" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want forbidigo findings, got $n"; exit 1; }
echo "$out" | grep -q "otelpgx.WithIncludeQueryParameters" || { echo "lint-fixture: otelpgx.WithIncludeQueryParameters was not reported"; exit 1; }
echo "$out" | grep -q "workflow.WithDataConverter" || { echo "lint-fixture: workflow.WithDataConverter was not reported"; exit 1; }
echo "lint-fixture: ok, $n forbidigo findings"
n=$(echo "$out" | grep -c "(depguard)$" || true)
test "$n" -eq 1 || { echo "lint-fixture: expected 1 depguard finding, got $n"; exit 1; }
echo "$out" | grep -q "identityport.go:.*lestrrat-go/jwx" || { echo "lint-fixture: the jwx import outside the adapter was not reported"; exit 1; }
echo "lint-fixture: ok, $n depguard finding"

want=6
code=0; out=$(NAVALINT_TAGS=lintfixture sh scripts/ci/navalint.sh ./internal/platform/lintfixture/ 2>&1) || code=$?
echo "$out"
test "$code" -ne 0 || { echo "lint-fixture: navalint passed the setconfig fixture"; exit 1; }
n=$(echo "$out" | grep -c "outside internal/tenancy: scope the transaction with tenancy.InTenantTx$" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want setconfig findings, got $n"; exit 1; }
for f in setconfig_integration.go setconfig_temporalstack.go; do
    echo "$out" | grep -q "$f:.*outside internal/tenancy" || { echo "lint-fixture: navalint did not check the build-tagged $f"; exit 1; }
done
echo "lint-fixture: ok, $n setconfig findings"

want=4
n=$(echo "$out" | grep -c "identityport.go:.*outside internal/identity/keycloak" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want idpclaims findings, got $n"; exit 1; }
echo "$out" | grep -q 'Keycloak claim "organization"' || { echo "lint-fixture: the organization claim read was not reported"; exit 1; }
echo "$out" | grep -q "Keycloak admin API" || { echo "lint-fixture: the admin API path was not reported"; exit 1; }
echo "lint-fixture: ok, $n idpclaims findings"

want=4
n=$(echo "$out" | grep -c "guarderrtype.go:.*outside internal/workerguard: only the worker guard builds this failure type (AR-044)$" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want guarderrtype findings, got $n"; exit 1; }
echo "$out" | grep -q "workerguard.ErrTypeTenantGuard used outside" || { echo "lint-fixture: the ErrTypeTenantGuard reference was not reported"; exit 1; }
echo "$out" | grep -q 'failure type "nava.WorkflowRefNotFound"' || { echo "lint-fixture: the nava.WorkflowRefNotFound literal was not reported"; exit 1; }
echo "lint-fixture: ok, $n guarderrtype findings"

# workflowcheck (D-3-23): exactly the 1 planted map range in workflow code.
code=0; out=$(GOFLAGS=-tags=lintfixture sh scripts/ci/workflowcheck.sh ./internal/platform/lintfixture/ 2>&1) || code=$?
echo "$out"
n=$(echo "$out" | grep -c "is non-deterministic, reason: iterates over map$" || true)
test "$n" -eq 1 || { echo "lint-fixture: expected 1 workflowcheck finding, got $n"; exit 1; }
echo "$out" | grep -q "workflowmaprange.go:.*MapOrderWorkflow is non-deterministic" || { echo "lint-fixture: the planted map range was not reported"; exit 1; }
echo "lint-fixture: ok, $n workflowcheck finding"

# ADR-002-E1: api.go carries one line-scoped waiver, not a file exclusion, so
# the raw http.Client ban must still fire in that very file. Append a raw
# client use to a copy-restored api.go and expect exactly that finding.
api=internal/server/api.go
bak=$(mktemp)
cp "$api" "$bak"
trap 'cp "$bak" "$api"; rm -f "$bak"' EXIT
printf '
// LintFixtureRawClient exists only while lint-fixture.sh runs.
func LintFixtureRawClient() *http.Client { return http.DefaultClient }
' >> "$api"
grep -q '"net/http"' "$api" || sed -i '0,/^import (/s//import (\n\t"net\/http"/' "$api"
code=0; out=$(${GOLANGCI_LINT:-golangci-lint} run ./internal/server/ 2>&1) || code=$?
echo "$out"
echo "$out" | grep -q "api.go:.*use telemetry.NewOutboundClient" || { echo "lint-fixture: the http.Client ban did not fire in $api"; exit 1; }
echo "lint-fixture: ok, http.Client ban fires in $api"
