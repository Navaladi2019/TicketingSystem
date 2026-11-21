#!/bin/sh
# S30 acceptance walk (TICK-944, PLAN-011 P11-12). Run from the repository root
# on a fresh, healthy stack of this checkout's images (README.md in this directory):
#   export NAVA_IMAGE_TAG=$(sh scripts/acceptance/image-tag.sh)
#   sh scripts/acceptance/fresh-stack.sh && sh docs/acceptance/S30/run.sh
# Seeds tenants A and B, shows how tenant A's incident ACL is stored before
# and after, runs the Bruno folders `S30 composable access` and `S10 access`
# (AC5: the S10 answers are unchanged), reads
# authz_policy_invalid_total, and ends with the D6 re-check. Prints no token
# or password. Exit status is non-zero on any failure.
set -eu
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1
step() { printf '\n== %s\n' "$*"; }
# Git Bash would rewrite /run/... in the container command.
psql_ro() {
  MSYS_NO_PATHCONV=1 docker compose exec -T postgres sh -c \
    'PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -v ON_ERROR_STOP=1 -U pgadmin -d nava "$@"' psql "$@"
}
stored() {
  psql_ro -c "BEGIN READ ONLY;
SELECT a.tenant_id = '$A' AS tenant_a, a.version, jsonb_typeof(a.rules) AS stored_as, a.published_by IS NULL AS seeded_default
  FROM acl_versions a JOIN entity_definitions d ON d.id = a.entity_definition_id
 WHERE d.key = 'incident' AND a.tenant_id IN ('$A', '$B') ORDER BY 1 DESC, 2;
ROLLBACK;"
}
metric() {
  MSYS_NO_PATHCONV=1 docker compose exec -T nava-core wget -qO- http://127.0.0.1:9090/metrics |
    grep -E '^authz_(policy_invalid|read_branches_fallback)_total' || echo "(no authz_policy_invalid_total or authz_read_branches_fallback_total series yet: zero)"
}

step "images: every nava/* container on this checkout's tag (TICK-1021)"
sh scripts/acceptance/check-images.sh

step "readyz (nava-core admin listener)"
MSYS_NO_PATHCONV=1 docker compose exec -T nava-core wget -qO- http://127.0.0.1:9090/readyz; echo

step "seed tenants A and B"
sh docs/acceptance/guide/tools/seed.sh | tail -3

step "the incident ACL as stored (array = format 1, object = format 2; seeded as format 2 since TICK-939)"
stored

step "authz_policy_invalid_total before the walk"
metric

step "Bruno folder S30 composable access (AC1 to AC5 and the isolation rows)"
sh docs/acceptance/guide/tools/api-test.sh S30

step "after: tenant A's versions, all format 2; tenant B unchanged at version 1"
stored

step "AC5: the S10 folder, written for the S10 rules, on the same stack"
sh docs/acceptance/guide/tools/api-test.sh S10

step "authz_policy_invalid_total after the walk (R7: stays at zero)"
metric

step "D6, format 1: the PLAN-011 SQL (newest format-1 documents only)"
psql_ro -f - <docs/acceptance/S30/d6.sql

step "D6, every newest document: authz.ReadBranchBound (tools/acceptance/acld6)"
psql_ro -At -F '|' -f - <docs/acceptance/S30/d6-docs.sql | go run ./tools/acceptance/acld6
