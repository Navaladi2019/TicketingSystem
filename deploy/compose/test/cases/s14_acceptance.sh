#!/bin/sh
# stack-case: job=stack order=28 mode=serial
# P5-20 (TICK-437): the S14 story's acceptance steps on the running stack.
#   1. tools/seed.sh: tenants A and B, the auditors groups, and incident,
#      request and change at version 1, written straight into
#      entity_definition_versions with no audit entry, as before S14.
#   2. audit_backfill_s8() (P5-15), run as nava_migrator: the same call the S14
#      migration made and migration B (P5-32) makes again. It files every
#      version that has no entry yet, so the seeded versions become the
#      "published before this release" entries AC2 reads. No nava_migrator
#      policy may be left after it (R26.3).
#   3. The Bruno folder "S14 audit" (S14A), headless: AC1 (CI-A11) one timeline
#      with the publish and both record changes in order, AC2 the backfilled
#      versions, AC3 agent 403 / own history / tenant B 404, AC4
#      config_secret_in_content naming the path only.
# The fuller upgrade path (a pre-S14 stack upgraded in place) is the kit's
# docs/acceptance/S14/tools/upgrade-setup.sh. Serial, after s14_audit (order
# 27), whose "reading writes nothing" check must not see this folder's writes.
# Needs Node.js (npx) on the host.
set -eu
. "$(dirname "$0")/../case-env.sh"
GUIDE=docs/acceptance/guide

command -v npx >/dev/null 2>&1 || fail "npx not found: install Node.js (verified with 20 and 22)"

sh "$GUIDE/tools/seed.sh" || fail "tools/seed.sh failed"
pass "seeded"

added=$(psql_super nava "SET ROLE nava_migrator; SELECT audit_backfill_s8();" | tr -d '\r ' | tail -n 1)
case $added in ''|*[!0-9]*) fail "audit_backfill_s8() as nava_migrator did not return a count: $added" ;; esac
pass "audit_backfill_s8() filed $added version(s) that had no entry"

left=$(psql_super nava "SELECT count(*) FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles)" | tr -d '\r ')
[ "$left" = 0 ] || fail "nava_migrator policies left after the backfill: $left"
pass "no nava_migrator policy outlives the backfill"

bruno=0
sh "$GUIDE/tools/api-test.sh" S14A || bruno=$?
[ "$bruno" -eq 0 ] || fail "Bruno folder S14 audit (S14A) failed (bru exit $bruno); see the bru output above"
pass "Bruno folder S14 audit (S14A): every acceptance step passed"
