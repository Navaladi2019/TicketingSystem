#!/bin/sh
# The lint rules must report exactly their findings on the fixtures in
# internal/platform/lintfixture: a partly flagged or non-compiling fixture
# fails.
#   forbidigo: 5 raw HTTP client uses (T04, Security Reviewer, TICK-55) and
#              1 otelpgx.WithIncludeQueryParameters (T07, TICK-94 item 4).
#   navalint setconfig: 2 tenant settings outside internal/tenancy (T07).
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

want=2
code=0; out=$(sh scripts/ci/navalint.sh -tags lintfixture ./internal/platform/lintfixture/ 2>&1) || code=$?
echo "$out"
test "$code" -ne 0 || { echo "lint-fixture: navalint passed the setconfig fixture"; exit 1; }
n=$(echo "$out" | grep -c "outside internal/tenancy: scope the transaction with tenancy.InTenantTx$" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want setconfig findings, got $n"; exit 1; }
echo "lint-fixture: ok, $n setconfig findings"
