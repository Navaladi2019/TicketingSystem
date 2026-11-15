#!/bin/sh
# stack-case: job=stack order=30 mode=serial
# The TypeScript SDK against the running stack (TICK-829, PLAN-008 P8-24; ADR-007 D10,
# RFC-017 CI-E8 for the SDK): sdk/ts/test/integration/ through make sdk-integration.
#   1. tools/seed.sh: tenants A and B (idempotent; run alone, the case needs no earlier case).
#   2. The Node tests, as Alice, tenant A's admin and Bob with password-grant tokens: list
#      across two pages, get, create replayed with one Idempotency-Key, the 401 retry,
#      stale version (VersionConflict), invalid body (record_invalid), another tenant's id
#      (404), a forced 429 on tenant B (RateLimited.retryAfter).
#   3. In Postgres: the replayed create has exactly one audit_log row (CI-E8).
# Serial: it writes records and spends tenant B's rate-limit bucket. Full tier, like the other
# story cases (s9_records): the daily run on main, ci:full and a manual run (TICK-652); the
# smoke set stays as TestRealSmokeTier pins it. Needs Node.js, the version in sdk/ts/.nvmrc
# (ci.yml sets it up in the stack job).
set -eu
. "$(dirname "$0")/../case-env.sh"
A=00000000-0000-4000-8000-0000000000a1

command -v npm >/dev/null 2>&1 || fail "npm not found: install Node.js $(cat sdk/ts/.nvmrc)"

sh docs/acceptance/guide/tools/seed.sh || fail "tools/seed.sh failed"
pass "seeded"

out=$(mktemp)
trap 'rm -f "$out"' EXIT
NAVA_CORE_URL="http://127.0.0.1:${NAVA_CORE_PORT:-18080}" \
NAVA_KEYCLOAK_URL="http://127.0.0.1:${KEYCLOAK_PORT:-18081}" \
SDK_IT_OUT="$out" \
    make --no-print-directory sdk-integration || fail "SDK integration tests failed; see the node --test output above"
pass "SDK integration tests"

id=$(tr -d '\r\n ' <"$out")
printf '%s\n' "$id" | grep -Eq '^[0-9a-f-]{36}$' || fail "CI-E8: the tests wrote no record id (got '$id')"
n=$(psql_super nava "SELECT count(*) FROM audit_log WHERE tenant_id = '$A' AND subject_type = 'record' AND subject_id = '$id'" | tr -d '\r ')
[ "$n" = 1 ] || fail "CI-E8: the replayed create has $n audit_log rows, want 1"
pass "CI-E8: the replayed create has one audit_log row"
