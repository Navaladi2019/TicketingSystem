#!/bin/sh
# stack-case: job=stack order=24 mode=serial
# Story S8 acceptance (TICK-333, kit TICK-346) on the running stack:
#   1. tools/seed.sh: tenants A and B, tenant A's admins group with
#      admin@tenant-a.test, incident/request/change at version 1 in both.
#   2. The Bruno folder "S8 entity types" (AC1-AC4, and S8.19 onboards a
#      tenant through nava-onboard), headless, with the pinned Bruno CLI.
#   3. AC5 in the database: every tenant onboarded by S8.19 (email domain
#      s8-*.test) starts with exactly incident, request and change, each at
#      version 1 ('initial', no publisher), with the schema bytes of the
#      template: the same schema_sha256 as tenant A's version 1 of that type,
#      which S8.6 checked against the template file byte for byte.
# Serial: it writes tenants, Keycloak groups and entity types, and opens the
# acceptance tunnels (closed again on exit). Needs Node.js (npx) on the host.
set -eu
. "$(dirname "$0")/../case-env.sh"
A=00000000-0000-4000-8000-0000000000a1
GUIDE=docs/acceptance/guide

command -v npx >/dev/null 2>&1 || fail "npx not found: install Node.js (verified with 20 and 22)"
printf 'node %s\n' "$(node --version)"

sh "$GUIDE/tools/seed.sh" || fail "tools/seed.sh failed"
trap 'sh "$GUIDE/tools/tunnels.sh" down >/dev/null 2>&1 || true' EXIT
sh "$GUIDE/tools/tunnels.sh" up || fail "tools/tunnels.sh up failed"
pass "seeded; tunnels up"

sh "$GUIDE/tools/api-test.sh" S8 || fail "Bruno folder S8 entity types failed (AC1-AC4); see the bru output above"
pass "Bruno folder S8 entity types: every check passed (AC1-AC4)"

tenants=$(psql_super nava "SELECT DISTINCT tenant_id FROM tenant_email_domain WHERE domain LIKE 's8-%.test' ORDER BY 1")
[ -n "$tenants" ] || fail "AC5: no tenant with an s8-*.test domain; S8.19 did not onboard one"
for t in $tenants; do
    t=$(printf '%s' "$t" | tr -d '\r')
    # One row per seeded type: key|current_version|versions|version|compat|published_by null|same sha as A's v1
    got=$(psql_super nava "
SELECT d.key || '|' || d.current_version || '|' || count(v.*) || '|' || min(v.version) || '|' || min(v.compat)
       || '|' || bool_and(v.published_by IS NULL) || '|' || bool_and(v.schema_sha256 = a.schema_sha256)
FROM entity_definitions d
JOIN entity_definition_versions v ON v.tenant_id = d.tenant_id AND v.definition_id = d.id
LEFT JOIN (SELECT ad.key, av.schema_sha256 FROM entity_definitions ad
           JOIN entity_definition_versions av ON av.tenant_id = ad.tenant_id AND av.definition_id = ad.id AND av.version = 1
           WHERE ad.tenant_id = '$A') a ON a.key = d.key
WHERE d.tenant_id = '$t'
GROUP BY d.key, d.current_version ORDER BY d.key" | tr -d '\r')
    want='change|1|1|1|initial|true|true
incident|1|1|1|initial|true|true
request|1|1|1|initial|true|true'
    [ "$got" = "$want" ] || fail "AC5: tenant $t: want
$want
got
$got"
    pass "AC5: newly onboarded tenant $t has change, incident and request at version 1 (initial, template schema bytes)"
done
