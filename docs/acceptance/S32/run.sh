#!/bin/sh
# S32 acceptance walk (TICK-958, PLAN-011 P11-24). Run from the repository root
# on a fresh, healthy stack (README.md in this directory):
#   docker compose down -v && docker compose up -d --build --wait && sh docs/acceptance/S32/run.sh
# Seeds tenants A and B and tenant B's admin, checks that no other incident
# workflow runs in tenant A, runs the Bruno folder `S32 workflow policy`, then
# shows from the database that the AC2 run was waiting before the policy was
# narrowed, the refused-reference audit entries per tenant, and the worker's
# authz_policy_ref_missing_total. Prints no token or password. Exit status is
# non-zero on any failure.
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
metric() {
  MSYS_NO_PATHCONV=1 docker compose exec -T nava-worker wget -qO- http://127.0.0.1:9090/metrics |
    grep -E '^authz_policy_ref_missing_total' || echo "(no authz_policy_ref_missing_total series yet: zero)"
}

step "readyz (nava-core admin listener)"
MSYS_NO_PATHCONV=1 docker compose exec -T nava-core wget -qO- http://127.0.0.1:9090/readyz; echo

step "seed tenants A and B"
sh docs/acceptance/guide/tools/seed.sh | tail -3

step "seed admin@tenant-b (the S18 kit's helper), then one group sync"
node docs/acceptance/S18/tools/kit.mjs admin-b
MSYS_NO_PATHCONV=1 docker compose run --rm --no-deps nava-groupsync --once >/dev/null
echo "group sync: done"

step "tenant A's incident workflows (must be s32_route or none: every current one fires on a create)"
others=$(psql_ro -At -c "BEGIN READ ONLY;
SELECT count(*) FROM workflow_definitions w JOIN entity_definitions d ON d.tenant_id = w.tenant_id AND d.id = w.entity_definition_id
 WHERE w.tenant_id = '$A' AND d.key = 'incident' AND w.key <> 's32_route';
ROLLBACK;" | grep -E '^[0-9]+$')
echo "other incident workflows in tenant A: $others"
[ "$others" = 0 ] || { echo "run.sh: tenant A has other incident workflows; use a fresh stack" >&2; exit 1; }

step "authz_policy_ref_missing_total before the walk"
metric

step "Bruno folder S32 workflow policy (AC1 to AC4, isolation, restore)"
sh docs/acceptance/guide/tools/api-test.sh S32

step "AC2: the run started before the narrowing and checked after it (workflow_ref, acl_versions)"
# One row per run on the newest AC2 incident: run_started|policy_narrowed|run_closed|started_before|closed_after.
# The narrowing is the agents_write version published within the 20 s timer after the create.
ac2=$(psql_ro -At -F '|' -c "BEGIN READ ONLY;
WITH rec AS (
  SELECT id, created_at FROM records WHERE tenant_id = '$A' AND data->>'short_description' LIKE 'S32 % AC2 %'
   ORDER BY created_at DESC LIMIT 1),
narrowed AS (
  SELECT max(a.published_at) AS at FROM acl_versions a JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.entity_definition_id
   WHERE a.tenant_id = '$A' AND d.key = 'incident'
     AND a.rules @> '{\"policies\": [{\"name\": \"agents_write\"}]}'
     AND a.published_at < (SELECT created_at FROM rec) + interval '20 seconds'
     AND a.published_at > (SELECT created_at FROM rec))
SELECT w.created_at, n.at, w.closed_at,
       coalesce(w.created_at < n.at, false), coalesce(w.closed_at > n.at, false)
  FROM workflow_ref w JOIN rec r ON w.record_id = r.id CROSS JOIN narrowed n
 WHERE w.tenant_id = '$A';
ROLLBACK;" | grep '|' || true)
echo "run_started|policy_narrowed|run_closed|started_before_narrowing|closed_after_narrowing"
echo "${ac2:-(no workflow_ref row for the AC2 incident)}"
rows=$(printf '%s' "$ac2" | grep -c '|' || true)
if [ "$rows" != 1 ] || [ "$(printf '%s' "$ac2" | cut -d'|' -f4,5)" != "t|t" ]; then
  echo "run.sh: AC2 FAILED: want exactly one run that started before the narrowing and closed after it ($rows rows)" >&2
  exit 1
fi
echo "AC2: PASS (the run was waiting when the policy was narrowed)"

step "refused policy references in the audit log, per tenant"
psql_ro -c "BEGIN READ ONLY;
SELECT tenant_id = '$A' AS tenant_a, action, actor_system, source, encode(content_sha256, 'hex') = encode(sha256('incident/does_not_exist'), 'hex') AS ac3,
       encode(content_sha256, 'hex') = encode(sha256('incident/tenant_b_only'), 'hex') AS ac4, count(*)
  FROM audit_log WHERE action = 'policy_ref_refused' AND tenant_id IN ('$A', '$B')
 GROUP BY 1, 2, 3, 4, 5, 6 ORDER BY 1 DESC, 5, 6;
ROLLBACK;"

step "authz_policy_ref_missing_total after the walk (AC3 and AC4 each add one or more)"
metric
