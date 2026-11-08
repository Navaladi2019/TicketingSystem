#!/bin/sh
# S16 acceptance (TICK-448, P5-43): walks the five S16 acceptance steps on a
# stack set up by docs/acceptance/S15/tools/upgrade-setup.sh, and writes the
# evidence to docs/acceptance/S16/evidence/.
#
#   1. Bruno folder S16 (S16 export), as auditor@tenant-a: the filtered
#      export, the audit/export entry, reseed 409 chain_alive, Bob's 403s and
#      the error answers.
#   2. tools/kit.mjs: the steps that need a file or a second request: the
#      filtered export verified offline (ok, then a deleted line and an edited
#      byte), a second concurrent export (409 export_in_progress), the CSV,
#      reseed, the S13 erasure samples, and tenant isolation of the export.
#
# Usage, from the repository root (Git Bash or any POSIX shell), Node.js 22:
#   sh docs/acceptance/S16/tools/run-kit.sh
# Takes about 5 minutes; most of it is writing the large incidents for the
# concurrent-export step (S16_LOAD, default 600).
set -eu
cd "$(dirname "$0")/../../../.."
if [ -f .env ]; then set -a; . ./.env; set +a; fi
EV=docs/acceptance/S16/evidence
mkdir -p "$EV"
# The raw Bruno reports hold dev-only tokens: never leave them behind (TICK-627).
trap 'rm -f "$EV"/.bruno-*.json' EXIT
CORE_PORT=${NAVA_CORE_PORT:-18080}
KC_PORT=${KEYCLOAK_PORT:-18081}
ADMIN_PORT=${ACCEPT_ADMIN_PORT:-19090}
FAILED=

echo "s16: build the offline verifier"
go build -o bin/nava$(go env GOEXE) ./cmd/nava

echo "s16: Bruno folder S16"
# Bruno (Node on Windows) needs Git Bash to convert its report paths.
env -u MSYS_NO_PATHCONV sh docs/acceptance/guide/tools/api-test.sh S16 \
  --env-var "core_url=http://127.0.0.1:$CORE_PORT" \
  --env-var "keycloak_url=http://127.0.0.1:$KC_PORT" \
  --env-var "keycloak_issuer_url=${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}" \
  --env-var "onboard_url=http://127.0.0.1:${ACCEPT_ONBOARD_PORT:-18082}" \
  --env-var "worker_admin_url=http://127.0.0.1:$((ADMIN_PORT + 4))" || FAILED="$FAILED S16"
cp docs/acceptance/guide/reports/S16.json "$EV/.bruno-S16.json"
# One evidence file per request, no credentials, and run-summary.json.
node docs/acceptance/S15/tools/evidence.mjs "$EV" || FAILED="$FAILED evidence"

echo "s16: kit.mjs"
node docs/acceptance/S16/tools/kit.mjs || FAILED="$FAILED kit"

if [ -n "$FAILED" ]; then echo "s16: FAILED:$FAILED; evidence in $EV" >&2; exit 1; fi
echo "s16: PASS; evidence in $EV"
