#!/bin/sh
# S31 acceptance walk (TICK-953, PLAN-011 P11-20). Run from the repository root
# on a fresh, healthy stack (README.md in this directory):
#   docker compose down -v && docker compose up -d --build --wait && sh docs/acceptance/S31/run.sh
# Seeds tenants A and B (the realm fixture already holds S31's directory: tenant
# A's /ou/EMEA, /ou/EMEA/Germany, /ou/APAC, agent-emea and four requesters),
# runs the Bruno folder `S31 region` (AC1, AC2, AC5, isolation, then the
# directory changes of AC3 and AC4 in Keycloak), runs one group sync and shows
# its result and the org-unit tree before and after, runs `S31 region after
# sync` (AC3, AC4), then shows from the database the unit each S31 ticket
# kept, the metrics behind S31's three alerts, and all three alerts as
# Prometheus (profile operator, started by this script) has them loaded. Prints no
# token or password. Exit status is non-zero on any failure.
set -eu
if [ -f .env ]; then set -a; . ./.env; set +a; fi
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1
step() { printf '\n== %s\n' "$*"; }
# Git Bash would rewrite /run/... in the container command.
psql_ro() {
  MSYS_NO_PATHCONV=1 docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -v ON_ERROR_STOP=1 -U pgadmin -d nava "$@"' psql "$@"
}
# Tenant A's org-unit tree: path, live or tombstoned, closure rows below it
# (itself included), members (primary marked *).
tree() {
  psql_ro -c "BEGIN READ ONLY;
WITH RECURSIVE t AS (
  SELECT id, parent_id, name::text AS path FROM org_units WHERE tenant_id = '$A' AND parent_id IS NULL
  UNION ALL
  SELECT u.id, u.parent_id, t.path || '/' || u.name FROM org_units u JOIN t ON u.parent_id = t.id WHERE u.tenant_id = '$A')
SELECT t.path, CASE WHEN u.deleted_at IS NULL THEN 'live' ELSE 'tombstoned' END AS state,
       (SELECT count(*) FROM org_unit_closure c WHERE c.tenant_id = '$A' AND c.ancestor_id = t.id) AS closure_rows,
       coalesce((SELECT string_agg(split_part(p.email, '@', 1) || CASE WHEN m.is_primary THEN '*' ELSE '' END, ', ' ORDER BY p.email)
                   FROM org_unit_members m JOIN principals p ON p.tenant_id = m.tenant_id AND p.id = m.principal_id
                  WHERE m.tenant_id = '$A' AND m.org_unit_id = t.id), '') AS members
  FROM t JOIN org_units u ON u.tenant_id = '$A' AND u.id = t.id
 ORDER BY t.path;
ROLLBACK;"
}

step "readyz (nava-core admin listener)"
MSYS_NO_PATHCONV=1 docker compose exec -T nava-core wget -qO- http://127.0.0.1:9090/readyz; echo

step "seed tenants A and B (and one group sync)"
sh docs/acceptance/guide/tools/seed.sh | tail -3

step "fresh-stack check: EMEA/Germany is live and no S31 ticket exists yet"
fresh=$(psql_ro -At -c "BEGIN READ ONLY;
SELECT (SELECT count(*) FROM org_units WHERE tenant_id = '$A' AND name = 'Germany' AND deleted_at IS NULL) || '|' ||
       (SELECT count(*) FROM records WHERE tenant_id = '$A' AND data->>'short_description' LIKE 'S31 %');
ROLLBACK;" | grep '|')
echo "live Germany units|S31 tickets: $fresh"
[ "$fresh" = "1|0" ] || { echo "run.sh: not a fresh stack (want 1|0); docker compose down -v first" >&2; exit 1; }

step "org-unit tree of tenant A, before (* = primary unit)"
tree

step "AC5's unit ids: tenant A's EMEA and tenant B's eng/platform (no API lists org units)"
unit() {
  psql_ro -At -c "BEGIN READ ONLY;
SELECT u.id FROM org_units u LEFT JOIN org_units p ON p.tenant_id = u.tenant_id AND p.id = u.parent_id
 WHERE u.tenant_id = '$1' AND u.name = '$2' AND coalesce(p.name, '') = '$3' AND u.deleted_at IS NULL;
ROLLBACK;" | grep -E '^[0-9a-f-]{36}$'
}
a_unit=$(unit "$A" EMEA '')
b_unit=$(unit "$B" platform eng)
echo "tenant A EMEA: $a_unit"
echo "tenant B eng/platform: $b_unit"

step "Bruno folder S31 region (AC1, AC2, AC5, isolation, directory changes)"
sh docs/acceptance/guide/tools/api-test.sh S31 --env-var "s31_a_unit=$a_unit" --env-var "s31_b_unit=$b_unit"

step "AC3: one group sync applies the directory changes (EMEA/Germany deleted, requester-apac moved)"
# The cycle logs JSON; keep tenant A's applied line and the cycle result.
out=$(MSYS_NO_PATHCONV=1 docker compose run --rm --no-deps nava-groupsync --once 2>&1) || {
  printf '%s\n' "$out" >&2; echo "run.sh: AC3 FAILED: the group sync run failed" >&2; exit 1; }
printf '%s\n' "$out" | grep -E '"msg":"(group sync applied|nava-groupsync --once done|group sync failed)' | grep -E "$A|--once done|failed" |
  sed -E 's/"(time|trace_id|span_id)":"[^"]*",?//g'
printf '%s\n' "$out" | grep -q '"msg":"nava-groupsync --once done".*"failed":0' || {
  echo "run.sh: AC3 FAILED: the sync reported a failed tenant" >&2; exit 1; }
echo "group sync: exit 0, failed 0"

step "org-unit tree of tenant A, after"
tree

step "Bruno folder S31 region after sync (AC3, AC4)"
sh docs/acceptance/guide/tools/api-test.sh S31-after

step "AC4 and AC3 from the database: the unit each S31 ticket kept"
psql_ro -c "BEGIN READ ONLY;
SELECT r.number, split_part(p.email, '@', 1) AS requester, coalesce(u.name, '(none)') AS ticket_unit,
       CASE WHEN u.id IS NULL THEN '' WHEN u.deleted_at IS NULL THEN 'live' ELSE 'tombstoned' END AS unit_state,
       coalesce((SELECT string_agg(ru.name, ', ') FROM org_unit_members m JOIN org_units ru ON ru.tenant_id = m.tenant_id AND ru.id = m.org_unit_id
                  WHERE m.tenant_id = r.tenant_id AND m.principal_id = r.requester_id AND m.is_primary), '(none)') AS requester_unit_now,
       r.data->>'short_description' AS short_description
  FROM records r
  JOIN principals p ON p.tenant_id = r.tenant_id AND p.id = r.requester_id
  LEFT JOIN org_units u ON u.tenant_id = r.tenant_id AND u.id = r.org_unit_id
 WHERE r.tenant_id = '$A' AND r.data->>'short_description' LIKE 'S31 %'
 ORDER BY r.created_at;
ROLLBACK;"
moved=$(psql_ro -At -c "BEGIN READ ONLY;
SELECT string_agg(u.name, ',' ORDER BY r.created_at) FROM records r
  JOIN principals p ON p.tenant_id = r.tenant_id AND p.id = r.requester_id
  JOIN org_units u ON u.tenant_id = r.tenant_id AND u.id = r.org_unit_id
 WHERE r.tenant_id = '$A' AND p.email = 'requester-apac@tenant-a.test' AND r.data->>'short_description' LIKE 'S31 %';
ROLLBACK;" | grep -v '^$' | grep -v -E '^(BEGIN|ROLLBACK)$')
echo "requester-apac's tickets, oldest first: $moved"
[ "$moved" = "APAC,EMEA" ] || { echo "run.sh: AC4 FAILED: want APAC,EMEA" >&2; exit 1; }
echo "AC4: PASS (the old ticket stays in APAC, the new one is in EMEA)"

step "metrics behind the three S31 alerts (nava-groupsync and nava-worker admin listeners)"
# nava-groupsync's own cycles (it syncs at start and every NAVA_GROUPSYNC_INTERVAL).
MSYS_NO_PATHCONV=1 docker compose exec -T nava-groupsync wget -qO- http://127.0.0.1:9090/metrics | grep -E '^groupsync_org_units_live_max[ {]' ||
  echo "(no groupsync_org_units_live_max sample yet: the service sets it on its first cycle, NAVA_GROUPSYNC_INTERVAL after start)"
MSYS_NO_PATHCONV=1 docker compose exec -T nava-groupsync wget -qO- http://127.0.0.1:9090/metrics | grep -E '^groupsync_org_closure_unrooted_units_total[ {]' ||
  echo "(no groupsync_org_closure_unrooted_units_total sample: no cycle of the service has added to it)"
MSYS_NO_PATHCONV=1 docker compose exec -T nava-worker wget -qO- http://127.0.0.1:9090/metrics | grep -E '^authz_under_estimate_age_seconds[ {]' ||
  echo "(no authz_under_estimate_age_seconds sample yet)"

step "S31 alerts loaded in Prometheus (profile operator, started here)"
MSYS_NO_PATHCONV=1 docker compose --profile operator up -d --wait prometheus >/dev/null 2>&1
MSYS_NO_PATHCONV=1 docker compose exec -T prometheus wget -qO- http://127.0.0.1:9091/api/v1/rules |
  node -e "let s='';process.stdin.on('data',d=>s+=d).on('end',()=>{const want=['GroupsyncOrgUnitsAboveCertified','GroupsyncOrgUnitsUnrooted','EntityRecordEstimatesStale'];
const rules=JSON.parse(s).data.groups.flatMap(g=>g.rules).filter(r=>want.includes(r.name));
rules.forEach(r=>console.log(r.name+': '+r.query+' (for '+r.duration+'s) state '+r.state));
const missing=want.filter(n=>!rules.some(r=>r.name===n));
if(missing.length||rules.length!==want.length){console.error('run.sh: want each of the three alerts loaded once; missing: '+(missing.join(', ')||'none')+', loaded: '+rules.length);process.exit(1);}})"
