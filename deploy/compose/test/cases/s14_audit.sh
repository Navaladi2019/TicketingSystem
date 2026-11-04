#!/bin/sh
# stack-case: job=stack order=27 mode=serial
# P5-17 (TICK-435): GET /api/v1/audit/entries on the running stack.
#   1. tools/seed.sh: tenants A and B.
#   2. The Bruno folder "S14 audit timeline", headless, with the pinned Bruno
#      CLI: tenant A's admin reads the timeline (newest first, stable pages,
#      domain filter, empty page), a member without audit.read gets 403, and
#      limit above 500, a bad page_token and no token are refused.
#   3. In the database: the run wrote nothing to audit_log (reading is not
#      audited), and audit_log_v0 is gone.
# Serial, after S9 (order 26), so the timeline has record entries to page.
# Needs Node.js (npx) on the host.
set -eu
. "$(dirname "$0")/../case-env.sh"
GUIDE=docs/acceptance/guide

command -v npx >/dev/null 2>&1 || fail "npx not found: install Node.js (verified with 20 and 22)"

sh "$GUIDE/tools/seed.sh" || fail "tools/seed.sh failed"
pass "seeded"

before=$(psql_super nava "SELECT count(*) FROM audit_log" | tr -d '\r ')

bruno=0
sh "$GUIDE/tools/api-test.sh" S14 || bruno=$?

after=$(psql_super nava "SELECT count(*) FROM audit_log" | tr -d '\r ')
[ "$before" = "$after" ] || fail "reading the timeline wrote audit entries: $before -> $after"
pass "reading the timeline wrote no audit entry"

gone=$(psql_super nava "SELECT coalesce(to_regclass('public.audit_log_v0')::text, 'gone')" | tr -d '\r ')
[ "$gone" = gone ] || fail "audit_log_v0 still exists: $gone"
pass "audit_log_v0 is dropped"

[ "$bruno" -eq 0 ] || fail "Bruno folder S14 audit timeline failed (bru exit $bruno); see the bru output above"
pass "Bruno folder S14 audit timeline: every check passed"
