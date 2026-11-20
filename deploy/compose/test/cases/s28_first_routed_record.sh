#!/bin/sh
# stack-case: job=stack order=61 mode=serial
# Story S28 M-X1a kit (TICK-879, kit P9-05 / TICK-884) on the running stack: the Bruno
# folder "S28 first routed record (M-X1a)" (S28.1 to S28.13), headless, with the pinned
# Bruno CLI. A tenant onboarded in this run (not a seeded one) gets its first admin through
# the local stand-in for the company login (S28.3-S28.5, Keycloak master admin), waits one
# group-sync cycle for the invite (up to 360 s at the default 5 min interval), publishes
# incident v2 from docs/handbook/examples/incident.json and the workflow s28_route
# (record.created -> PT1M -> state in_progress), creates an incident and polls up to 300 s
# for the source=workflow history entry. Reports go to
# docs/acceptance/guide/reports/S28.{junit.xml,html,json}; CI uploads them.
# Order 61: after s11b_workflows (59). It needs no seed: its tenant is its own. Serial: it
# onboards a tenant, writes Keycloak users and records, and opens the acceptance tunnels
# (closed again on exit). Needs Node.js and npm (the locked Bruno CLI, tools/bruno, TICK-1003).
set -eu
. "$(dirname "$0")/../case-env.sh"
GUIDE=docs/acceptance/guide

command -v npm >/dev/null 2>&1 || fail "npm not found: install Node.js (verified with 20 and 22)"

trap 'sh "$GUIDE/tools/tunnels.sh" down >/dev/null 2>&1 || true' EXIT
sh "$GUIDE/tools/tunnels.sh" up || fail "tools/tunnels.sh up failed"
pass "tunnels up"

bruno=0
sh "$GUIDE/tools/api-test.sh" S28 || bruno=$?
[ "$bruno" -eq 0 ] || fail "Bruno folder S28 failed (bru exit $bruno); see the bru output above and the s28-first-routed-record-report artifact"
pass "Bruno folder S28: every check passed (S28.1-S28.13), onboarding to a workflow-routed record"
