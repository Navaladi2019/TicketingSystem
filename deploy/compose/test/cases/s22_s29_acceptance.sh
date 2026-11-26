#!/bin/sh
# stack-case: job=stack order=70 mode=serial profile=operator
# Stories S22 (TICK-740) and S29 (TICK-915), kit TICK-754, on the running stack:
# the Bruno folders "S22 unflag and quota" and "S29 schema backfill", headless,
# with their seed, expectation and verify SQL (docs/acceptance/S22/README.md,
# docs/acceptance/S29/README.md). The same steps as
# docs/acceptance/S22/tools/run-s22-s29.sh, minus the fresh stack: S29's prep
# (its incidents), then S22, then S29's checks. Evidence goes to
# docs/acceptance/S22/evidence/ and docs/acceptance/S29/evidence/; the Bruno
# reports to docs/acceptance/guide/reports/S22.* and S29.*.
#
# Order 70: after every read-only and story case (s28 is 61, dbroles 62). It
# writes: the type s22_asset, a quota type, tenant A's tenant_limits row (10),
# about 2,000 incidents and two incident versions, and admin@tenant-b. Serial.
# profile=operator: S29.8 and S29.17 list workflow runs through the Temporal UI.
#
# Until every S22 and S29 feature task has merged into the branch, the case
# prints SKIP naming what is missing and passes, so it does not turn the
# feature PRs on TICK-740/search-finish red before the code it checks exists.
# Once every marker is present it always runs.
set -eu
. "$(dirname "$0")/../case-env.sh"
GUIDE=docs/acceptance/guide

missing=
ls db/migrations/*tenant_limits*.sql >/dev/null 2>&1 || missing="$missing tenant_limits-migration(TICK-916)"
ls db/migrations/*schema_backfill*.sql >/dev/null 2>&1 || missing="$missing schema_backfill-tables(TICK-916)"
grep -q 'backfill/failures' api/openapi.yaml 2>/dev/null || missing="$missing backfill-failures-route(TICK-919)"
grep -q 'operator/tenants/{tenant}/limits' api/openapi.yaml 2>/dev/null || missing="$missing limits-route(TICK-916)"
[ -f internal/fieldindex/backfill/remove.go ] || missing="$missing FieldIndexRemove(TICK-750)"
if git grep -q DataFilterTimeout -- '*.go' ':!*_test.go' 2>/dev/null; then missing="$missing scan-retired(TICK-752)"; fi
if [ -n "$missing" ]; then
  printf 'SKIP: s22_s29_acceptance: not every S22/S29 feature is on this branch yet:%s\n' "$missing"
  exit 0
fi

command -v npm >/dev/null 2>&1 || fail "npm not found: install Node.js (verified with 20 and 22)"
export COMPOSE_PROJECT_NAME="$PROJECT"

trap 'sh "$GUIDE/tools/tunnels.sh" down >/dev/null 2>&1 || true' EXIT
sh "$GUIDE/tools/tunnels.sh" up || fail "tools/tunnels.sh up failed"
pass "tunnels up"

node docs/acceptance/S18/tools/kit.mjs admin-b || fail "seeding admin@tenant-b failed"
compose run --rm --no-deps nava-groupsync --once || fail "one group sync cycle failed"
pass "admin@tenant-b seeded and synced"

S29_KEEP_STACK=1 S29_PHASE=prep sh docs/acceptance/S29/tools/run-kit.sh || fail "S29 prep failed (setup, seed, held, expectations); see the output above"
pass "S29 prep: 2,000 incidents without priority, 50 with it, expectations"
S22_KEEP_STACK=1 sh docs/acceptance/S22/tools/run-kit.sh || fail "S22 kit failed; see docs/acceptance/S22/evidence/ and the S22 Bruno report"
pass "S22 kit: criteria 1 to 4"
S29_KEEP_STACK=1 S29_PHASE=check sh docs/acceptance/S29/tools/run-kit.sh || fail "S29 kit failed; see docs/acceptance/S29/evidence/ and the S29 Bruno report"
pass "S29 kit: criteria 1 to 6"
