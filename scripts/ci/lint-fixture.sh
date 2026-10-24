#!/bin/sh
# The lint rules must report exactly their findings on the fixtures in
# internal/platform/lintfixture: a partly flagged or non-compiling fixture
# fails.
#   forbidigo: 5 raw HTTP client uses (T04, Security Reviewer, TICK-55) and
#              1 otelpgx.WithIncludeQueryParameters (T07, TICK-94 item 4).
#   depguard: 1 JWT library import outside internal/identity/keycloak (T15,
#              CI-11).
#   navalint setconfig: 2 tenant settings outside internal/tenancy (T07),
#              seen by both of navalint.sh's passes, plus 1 in an
#              integration-tagged and 1 in a temporalstack-tagged file, seen
#              only by the tagged pass `make lint` runs.
#   navalint idpclaims: 1 Keycloak claim read and 1 admin API path outside
#              internal/identity/keycloak (T15, CI-11), seen by both passes.
set -u
cd "$(dirname "$0")/../.."
want=6
code=0; out=$(${GOLANGCI_LINT:-golangci-lint} run --build-tags lintfixture ./internal/platform/lintfixture/ 2>&1) || code=$?
echo "$out"
test "$code" -eq 1 || { echo "lint-fixture: expected exit 1, got $code"; exit 1; }
n=$(echo "$out" | grep -c "(forbidigo)$" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want forbidigo findings, got $n"; exit 1; }
echo "$out" | grep -q "otelpgx.WithIncludeQueryParameters" || { echo "lint-fixture: otelpgx.WithIncludeQueryParameters was not reported"; exit 1; }
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
