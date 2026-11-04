#!/bin/sh
# stack-case: job=stack order=26 mode=serial
# Story S9 acceptance (TICK-335, kit TICK-359) on the running stack:
#   1. tools/seed.sh: tenants A and B with the incident type.
#   2. The Bruno folder "S9 records" (AC1-AC5), headless, with the pinned
#      Bruno CLI. The run id is fixed here so the database checks below can
#      find the run's rows. s9_fresh_a / s9_fresh_b are 1 when the tenant has
#      no incident yet, and only then does Bruno assert the exact INC0000001;
#      otherwise it checks the shape and that there is no gap.
#   3. In the database, for this run:
#      AC1  exactly one record for the retried create; numbers ^INC\d{7}$,
#           the second create's number right after the first.
#      AC3  the updated incident has two audit_log entries, create and
#           state_change (S9.9 moves it to in_progress; P5-13b, TICK-452),
#           both source=user, both by Alice, the record's created_by.
#      AC4  the refused create left no record and no audit entry.
# Serial: it writes records. Needs Node.js (npx) on the host.
set -eu
. "$(dirname "$0")/../case-env.sh"
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1
GUIDE=docs/acceptance/guide

command -v npx >/dev/null 2>&1 || fail "npx not found: install Node.js (verified with 20 and 22)"

sh "$GUIDE/tools/seed.sh" || fail "tools/seed.sh failed"
pass "seeded"

run=$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')
incidents() {
    psql_super nava "SELECT count(*) FROM records r JOIN entity_definitions d
        ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
        WHERE r.tenant_id = '$1' AND d.key = 'incident'" | tr -d '\r '
}
fresh_a=0; [ "$(incidents "$A")" = 0 ] && fresh_a=1
fresh_b=0; [ "$(incidents "$B")" = 0 ] && fresh_b=1
printf 'S9 run %s; fresh: tenant A %s, tenant B %s\n' "$run" "$fresh_a" "$fresh_b"

# The database checks run even when a Bruno check fails, so one run reports every criterion.
bruno=0
sh "$GUIDE/tools/api-test.sh" S9 --env-var "s9_run_id=$run" --env-var "s9_fresh_a=$fresh_a" \
    --env-var "s9_fresh_b=$fresh_b" || bruno=$?

# The run's tenant A incidents: number|short_description, in number order.
got=$(psql_super nava "SELECT r.number || '|' || (r.data->>'short_description') FROM records r
    JOIN entity_definitions d ON d.tenant_id = r.tenant_id AND d.id = r.entity_definition_id
    WHERE r.tenant_id = '$A' AND d.key = 'incident' AND r.data::text LIKE '%S9 $run %'
    ORDER BY r.number" | tr -d '\r')
printf '%s\n' "$got"
n1=$(printf '%s\n' "$got" | grep -c "|S9 $run printer on fire\$" || true)
[ "$n1" = 1 ] || fail "AC1: want exactly one record for the retried create, got $n1"
pass "AC1: the retried create stored exactly one record"
first=$(printf '%s\n' "$got" | grep "|S9 $run printer on fire\$" | cut -d'|' -f1)
second=$(printf '%s\n' "$got" | grep "|S9 $run second\$" | cut -d'|' -f1)
for num in $first $second; do
    printf '%s\n' "$num" | grep -Eq '^INC[0-9]{7}$' || fail "AC1: number $num is not INC and 7 digits"
done
[ -n "$second" ] || fail "AC4: the second create (S9.8) stored no record"
want=$(printf 'INC%07d' $(( $(printf '%s' "${first#INC}" | sed 's/^0*//;s/^$/0/') + 1 )))
[ "$second" = "$want" ] || fail "AC4: second number $second, want $want (a gap: something was created)"
pass "AC1, AC4: numbers $first and $second, INC and 7 digits, no gap"
if [ "$fresh_a" = 1 ]; then
    [ "$first" = INC0000001 ] || fail "AC1: fresh stack, want INC0000001, got $first"
    pass "AC1: fresh stack: INC0000001"
fi

bad=$(psql_super nava "SELECT (SELECT count(*) FROM records WHERE tenant_id = '$A' AND data::text LIKE '%S9 $run invalid%')
    || '|' || (SELECT count(*) FROM audit_log WHERE tenant_id = '$A' AND domain = 'record'
               AND coalesce(diff::text, '') LIKE '%S9 $run invalid%')" | tr -d '\r ')
[ "$bad" = "0|0" ] || fail "AC4: the refused create left records|audit entries: $bad"
pass "AC4: the refused create stored no record and no history"

hist=$(psql_super nava "SELECT string_agg(a.action || ',' || a.source || ',' || (a.actor_principal_id = r.created_by)::text, ' ' ORDER BY a.version_after)
    FROM audit_log a JOIN records r ON r.tenant_id = a.tenant_id AND a.subject_type = 'record' AND r.id = a.subject_id
    WHERE r.tenant_id = '$A' AND r.number = '$first'
      AND r.entity_definition_id = (SELECT id FROM entity_definitions WHERE tenant_id = '$A' AND key = 'incident')" | tr -d '\r')
[ "$hist" = "create,user,true state_change,user,true" ] || fail "AC3: audit entries of $first: want 'create,user,true state_change,user,true', got '$hist'"
alice=$(psql_super nava "SELECT count(*) FROM records r JOIN principals p ON p.tenant_id = r.tenant_id AND p.id = r.created_by
    WHERE r.tenant_id = '$A' AND r.number = '$first' AND p.email = 'alice@tenant-a.test'
      AND r.entity_definition_id = (SELECT id FROM entity_definitions WHERE tenant_id = '$A' AND key = 'incident')" | tr -d '\r ')
[ "$alice" = 1 ] || fail "AC3: $first was not created by alice@tenant-a.test"
pass "AC3: $first has two audit entries (create, state_change), source user, both by Alice"

[ "$bruno" -eq 0 ] || fail "Bruno folder S9 records failed (bru exit $bruno); see the bru output above"
pass "Bruno folder S9 records: every check passed (AC1-AC5)"
