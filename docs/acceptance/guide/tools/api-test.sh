#!/bin/sh
# Acceptance guide (TICK-331): runs the Bruno collection headless against the
# local stack (environment bruno/environments/local.bru) with the pinned Bruno
# CLI, and writes JUnit, HTML and JSON reports to docs/acceptance/guide/reports/
# (git-ignored). Exits non-zero when any check fails.
#
# Before a full run: stack up, tools/seed.sh, tools/tunnels.sh up, and
# `make demo-outbox DEDUPE_KEY=accept` twice (see prerequisites.html).
#
# Usage, from the repository root (Git Bash on Windows, or any POSIX shell):
#   sh docs/acceptance/guide/tools/api-test.sh            # the whole collection
#   sh docs/acceptance/guide/tools/api-test.sh S4         # one folder
#   sh docs/acceptance/guide/tools/api-test.sh S4 --env-var core_url=http://127.0.0.1:28080
# Arguments after the folder go to `bru run` unchanged.
#
# The reports hold the requests and responses, so they carry the dev-only seed
# passwords and short-lived tokens of this run (Authorization headers are
# left out). Do not attach them anywhere public.
set -eu
BRUNO_CLI_VERSION=4.3.0

GUIDE=$(cd "$(dirname "$0")/.." && pwd)
COLL="$GUIDE/bruno"
OUT="$GUIDE/reports"

target=.
name=all
if [ $# -gt 0 ] && [ "${1#-}" = "$1" ]; then
  if [ ! -d "$COLL/$1" ]; then
    echo "api-test: no folder $1 in $COLL; one of: $(cd "$COLL" && ls -d */ | tr -d / | tr '\n' ' ')" >&2
    exit 2
  fi
  target=$1
  name=$1
  shift
fi

command -v npx >/dev/null 2>&1 || { echo "api-test: npx not found; install Node.js (verified with 22)" >&2; exit 2; }
mkdir -p "$OUT"

# A stack on other host ports (make ci-isolated, the self-hosted CI runner,
# TICK-641) exports them; local.bru names the defaults. Explicit --env-var
# arguments come after these, so they still win.
envs=""
add() { [ -z "$2" ] || envs="$envs --env-var $1=$2"; }
add core_url "${NAVA_CORE_PORT:+http://127.0.0.1:$NAVA_CORE_PORT}"
add keycloak_url "${KEYCLOAK_PORT:+http://127.0.0.1:$KEYCLOAK_PORT}"
add keycloak_issuer_url "${KEYCLOAK_PUBLIC_URL:-}"
add onboard_url "${ACCEPT_ONBOARD_PORT:+http://127.0.0.1:$ACCEPT_ONBOARD_PORT}"
add temporal_ui_url "${ACCEPT_UI_PORT:+http://127.0.0.1:$ACCEPT_UI_PORT}"
if [ -n "${ACCEPT_ADMIN_PORT:-}" ]; then
  i=0
  for v in admin_url onboard_admin_url groupsync_admin_url relay_admin_url worker_admin_url purge_admin_url; do
    add "$v" "http://127.0.0.1:$((ACCEPT_ADMIN_PORT + i))"
    i=$((i + 1))
  done
fi

# Bruno resolves require('./lib/nava') and the environment from the
# collection root, so it runs from there.
cd "$COLL"
set +e
npx --yes "@usebruno/cli@$BRUNO_CLI_VERSION" run "$target" -r \
  --env local \
  --sandbox developer \
  --reporter-skip-headers Authorization \
  --reporter-junit "$OUT/$name.junit.xml" \
  --reporter-html "$OUT/$name.html" \
  --reporter-json "$OUT/$name.json" \
  $envs "$@"
rc=$?
set -e
echo "api-test: reports in $OUT/$name.{junit.xml,html,json}"
if [ "$rc" -ne 0 ]; then
  echo "api-test: FAILED (bru exit $rc)" >&2
  exit "$rc"
fi
echo "api-test: PASS"
