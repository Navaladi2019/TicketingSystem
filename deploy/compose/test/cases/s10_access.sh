#!/bin/sh
# stack-case: job=stack order=27 mode=serial
# Story S10 (TICK-336, follow-up TICK-643) on the running stack, on the default
# incident ACL:
#   1. tools/seed.sh: tenants A and B, tenant A's admins group, and the default
#      incident ACL (nava-onboard reconcile-default-acl).
#   2. The Bruno folder "S10 access" (S10.1 to S10.27), headless, with the pinned
#      Bruno CLI: the record list under the ACL (P3-34), then the access policy
#      GET/PUT and the explainer (P3-35). Reports go to
#      docs/acceptance/guide/reports/S10.{junit.xml,html,json}; CI uploads them.
# Order 27, after s9_records (26): s10_access sorts before s14_audit (27) by name.
# The folder needs no incident from S9: S10.17 creates its own.
# Serial: it publishes an ACL version and creates records. Needs Node.js (npx).
set -eu
. "$(dirname "$0")/../case-env.sh"
GUIDE=docs/acceptance/guide

command -v npx >/dev/null 2>&1 || fail "npx not found: install Node.js (verified with 20 and 22)"

sh "$GUIDE/tools/seed.sh" || fail "tools/seed.sh failed"
pass "seeded (default incident ACL published)"

bruno=0
sh "$GUIDE/tools/api-test.sh" S10 || bruno=$?
[ "$bruno" -eq 0 ] || fail "Bruno folder S10 access failed (bru exit $bruno); see the bru output above and the s10-access-report artifact"
pass "Bruno folder S10 access: every check passed (S10.1-S10.27)"
