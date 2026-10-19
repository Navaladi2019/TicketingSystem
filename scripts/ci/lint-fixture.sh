#!/bin/sh
# The forbidigo rule must report exactly 5 findings on the raw HTTP client
# fixtures (T04, Security Reviewer, TICK-55): a partly flagged or
# non-compiling fixture fails.
set -u
cd "$(dirname "$0")/../.."
want=5
code=0; out=$(${GOLANGCI_LINT:-golangci-lint} run --build-tags lintfixture ./internal/platform/lintfixture/ 2>&1) || code=$?
echo "$out"
test "$code" -eq 1 || { echo "lint-fixture: expected exit 1, got $code"; exit 1; }
n=$(echo "$out" | grep -c "(forbidigo)$" || true)
test "$n" -eq "$want" || { echo "lint-fixture: expected $want forbidigo findings, got $n"; exit 1; }
echo "lint-fixture: ok, $n forbidigo findings"
