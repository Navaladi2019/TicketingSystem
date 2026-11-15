#!/bin/sh
# stack-case: job=stack order=59 mode=serial
# Story S11b acceptance (TICK-337, kit P3-44 / TICK-713) on the running stack:
#   1. tools/seed.sh: tenants A and B with the incident type and tenant A's admins group.
#   2. The Bruno folder "S11b workflow definitions" (S11b.1 to S11b.30), headless, with
#      the pinned Bruno CLI. S11b.17 to S11b.30 are the story criteria: the admin publishes
#      s11b_demo (on record.created of an incident, wait PT1M, set state=in_progress), Alice's
#      incident is in_progress about a minute later with a source=workflow history row, v2
#      published while that run waits leaves it on v1, and Bob's tenant B incident is untouched.
#      The always-on nava-worker runs the interpreter with the production ports. Reports go to
#      docs/acceptance/guide/reports/S11b.{junit.xml,html,json}.
#   3. In the database, for this run's incidents (found by the run id in short_description):
#      AC2  the first incident has exactly one source=workflow audit entry and its run's
#           workflow_ref is closed.
#      AC3  the first incident's start row names the demo version S11b.18 published, the
#           second's the next one (S11b.21): each run started on the version current at
#           its create, and the first never moved.
#      AC4  tenant B's incident has no nava.interpreter start row, no workflow_ref and no
#           source=workflow entry, and tenant B has no nava.interpreter row naming it at all.
# Order 59: after the interpreter case (58), whose crash case retires its own definition, so
# the always-on nava-worker is back and the only nava-tq poller. Serial: it publishes and writes.
# Needs Node.js (npx).
set -eu
. "$(dirname "$0")/../case-env.sh"
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1
GUIDE=docs/acceptance/guide

command -v npx >/dev/null 2>&1 || fail "npx not found: install Node.js (verified with 20 and 22)"

sh "$GUIDE/tools/seed.sh" || fail "tools/seed.sh failed"
pass "seeded"

run=$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')
printf 'S11b run %s\n' "$run"

# The database checks run even when a Bruno check fails, so one run reports every criterion.
bruno=0
sh "$GUIDE/tools/api-test.sh" S11b --env-var "s11b_run_id=$run" || bruno=$?

rec() {
    psql_super nava "SELECT id FROM records WHERE tenant_id = '$1' AND data->>'short_description' = 'S11b $run $2'" | tr -d '\r '
}
i1=$(rec "$A" one); i2=$(rec "$A" two); b1=$(rec "$B" "tenant B")
[ -n "$i1" ] && [ -n "$i2" ] && [ -n "$b1" ] || fail "this run's incidents not found (one '$i1', two '$i2', tenant B '$b1')"

# The demo version each tenant A incident's run started on.
started_on() {
    psql_super nava "SELECT string_agg(v.version::text, ',' ORDER BY v.version) FROM outbox o
        JOIN workflow_definition_versions v ON v.tenant_id = o.tenant_id AND v.id = (o.payload->>'definition_version_id')::uuid
        JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
        WHERE o.tenant_id = '$A' AND o.kind = 'nava.interpreter' AND d.key = 's11b_demo'
          AND o.payload->>'record_id' = '$1'" | tr -d '\r '
}
v1=$(started_on "$i1"); v2=$(started_on "$i2")
printf 'demo versions started: first incident %s, second incident %s\n' "$v1" "$v2"
case $v1 in ''|*,*) fail "AC3: the first incident has demo start rows '$v1', want exactly one" ;; esac
[ "$v2" = "$((v1 + 1))" ] || fail "AC3: the second incident started on demo version '$v2', want $((v1 + 1)) (the v2 published while the first run waited)"
pass "AC3: the first incident's run started on demo version $v1, the second's on $v2"

wf=$(psql_super nava "SELECT (SELECT count(*) FROM audit_log WHERE tenant_id = '$A' AND subject_id = '$i1' AND source = 'workflow')
    || '|' || (SELECT count(*) FROM workflow_ref f JOIN outbox o ON o.tenant_id = f.tenant_id AND o.id = f.outbox_id
               JOIN workflow_definition_versions v ON v.tenant_id = o.tenant_id AND v.id = (o.payload->>'definition_version_id')::uuid
               JOIN workflow_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
               WHERE f.tenant_id = '$A' AND f.record_id = '$i1' AND d.key = 's11b_demo' AND f.closed_at IS NOT NULL)" | tr -d '\r ')
[ "$wf" = "1|1" ] || fail "AC2: first incident's source=workflow entries|closed demo runs = $wf, want 1|1"
pass "AC2: the first incident has one source=workflow entry and its demo run is closed"

iso=$(psql_super nava "SELECT (SELECT count(*) FROM outbox WHERE kind = 'nava.interpreter' AND payload->>'record_id' = '$b1')
    || '|' || (SELECT count(*) FROM workflow_ref WHERE record_id = '$b1')
    || '|' || (SELECT count(*) FROM audit_log WHERE tenant_id = '$B' AND subject_id = '$b1' AND source = 'workflow')
    || '|' || (SELECT count(*) FROM workflow_definitions WHERE tenant_id = '$B')" | tr -d '\r ')
[ "$iso" = "0|0|0|0" ] || fail "AC4: tenant B's incident: start rows|workflow refs|workflow entries|tenant B definitions = $iso, want 0|0|0|0"
pass "AC4: tenant B's incident started no workflow (no start row, no ref, no workflow write)"

[ "$bruno" -eq 0 ] || fail "Bruno folder S11b failed (bru exit $bruno); see the bru output above and the report"
pass "Bruno folder S11b: every check passed (S11b.1-S11b.30)"
