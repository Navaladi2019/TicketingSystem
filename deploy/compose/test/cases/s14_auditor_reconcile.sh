#!/bin/sh
# stack-case: job=stack order=29 mode=serial
# P5-22 (TICK-567): the upgrade reconcile on the running stack, against real
# Postgres and Keycloak. Tenant A is made to look like a tenant activated
# before S14: its auditors group (Keycloak group, groups row, pointer row) is
# removed. Then:
#   1. an auditor outside any auditors group is refused (403);
#   2. `nava-onboard reconcile-auditors A` twice: the group, the groups row and
#      the pointer are back, with no member, and the second run changes nothing;
#   3. auditor@tenant-a.test is added to the new group, one group sync cycle
#      runs, and GET /api/v1/audit/entries answers 200.
# Serial, after s14_acceptance (order 28), which seeds the groups this case
# removes. Needs the stack's seed (tools/seed.sh, run by that case).
set -eu
. "$(dirname "$0")/../case-env.sh"
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1
KC="http://127.0.0.1:${KEYCLOAK_PORT:-18081}"
CORE="http://127.0.0.1:${NAVA_CORE_PORT:-18080}"

kc_token=$(curl -fsS -X POST "$KC/realms/master/protocol/openid-connect/token" \
  -d grant_type=password -d client_id=admin-cli \
  --data-urlencode "username=${KEYCLOAK_ADMIN:-admin}" \
  --data-urlencode "password=${KEYCLOAK_ADMIN_PASSWORD:-dev-only-admin}" |
  sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
[ -n "$kc_token" ] || fail "no Keycloak admin token from $KC"
kc() { curl -fsS -H "Authorization: Bearer $kc_token" -H 'Content-Type: application/json' "$@"; }
first_id() { sed -n 's/^[^{]*{"id":"\([^"]*\)".*/\1/p' | head -n 1; }
count() { psql_super nava "$1" | tr -d '\r '; }
audit_code() { # <access token>
  curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $1" "$CORE/api/v1/audit/entries"
}
auditor_token() {
  curl -fsS -X POST "$KC/realms/nava/protocol/openid-connect/token" \
    -d grant_type=password -d client_id=nava-web -d scope=openid \
    -d username=auditor@tenant-a.test -d password=dev-only-auditor-a |
    sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p'
}
snapshot() {
  count "SELECT (SELECT count(*) FROM groups WHERE tenant_id = '$A' AND path = '/t/$A/auditors')::text || '/' ||
    (SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = '$A')::text || '/' ||
    (SELECT count(*) FROM group_members m JOIN groups g ON g.tenant_id = m.tenant_id AND g.id = m.group_id
       WHERE g.tenant_id = '$A' AND g.path = '/t/$A/auditors')::text || '/' ||
    (SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = '$B')::text"
}

# Make A a pre-S14 tenant: no Keycloak group, no groups row, no pointer.
old=$(kc "$KC/admin/realms/nava/group-by-path/t/$A/auditors" | first_id || true)
[ -z "$old" ] || kc -X DELETE "$KC/admin/realms/nava/groups/$old" >/dev/null
psql_super nava "DELETE FROM tenant_auditor_group WHERE tenant_id = '$A'; DELETE FROM groups WHERE tenant_id = '$A' AND path = '/t/$A/auditors'" >/dev/null
b_before=$(count "SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = '$B'")
[ "$(snapshot)" = "0/0/0/$b_before" ] || fail "tenant A still has an auditors group: $(snapshot)"
pass "tenant A has no auditors group, as before S14"

tok=$(auditor_token)
[ -n "$tok" ] || fail "no token for auditor@tenant-a.test"
code=$(audit_code "$tok")
[ "$code" = 403 ] || fail "auditor outside an auditors group got HTTP $code, want 403"
pass "an auditor without the group is refused (403)"

reconcile() { compose exec -T nava-onboard nava-onboard reconcile-auditors "$A"; }
reconcile || fail "reconcile-auditors (first run) failed"
first=$(snapshot)
[ "$first" = "1/1/0/$b_before" ] || fail "after the reconcile: groups/pointer/members/B-pointer = $first, want 1/1/0/$b_before"
[ "$(kc "$KC/admin/realms/nava/group-by-path/t/$A/auditors" | first_id)" ] || fail "Keycloak has no /t/$A/auditors after the reconcile"
pass "reconcile created the Keycloak group, the groups row and the pointer, with no member, and left B alone"

reconcile || fail "reconcile-auditors (second run) failed"
[ "$(snapshot)" = "$first" ] || fail "second reconcile changed rows: $first -> $(snapshot)"
pass "a second reconcile changes nothing"

# An auditor added afterwards reads the timeline.
group=$(kc "$KC/admin/realms/nava/group-by-path/t/$A/auditors" | first_id)
user=$(kc "$KC/admin/realms/nava/users?username=auditor%40tenant-a.test&exact=true" | sed 's/^\[//' | first_id)
[ -n "$group" ] && [ -n "$user" ] || fail "no auditors group or no auditor@tenant-a.test"
kc -X PUT "$KC/admin/realms/nava/users/$user/groups/$group" >/dev/null
compose run --rm --no-deps nava-groupsync --once >/dev/null || fail "group sync failed"
tok=$(auditor_token)
code=$(audit_code "$tok")
[ "$code" = 200 ] || fail "auditor added to the reconciled group got HTTP $code on GET /api/v1/audit/entries, want 200"
pass "an auditor added to the reconciled group reads GET /api/v1/audit/entries (200)"
