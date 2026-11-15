#!/bin/sh
# TICK-789: publishes the three example record types of the handbook tour
# (incident, loan_application, visa_application) into tenant A of a local
# stack, with their access policies and field masks, then creates one record
# of each through the API as admin@tenant-a.test.
#
# Needs a seeded stack: run docs/acceptance/guide/tools/seed.sh first (it
# creates tenant A, its admins group and the base `incident` type).
# Idempotent: a type that exists is kept (incident gets its extra field as
# version 2 once), an access policy is published only while the type has none
# of this script's rules, and each record is created under a fixed
# Idempotency-Key, so a re-run replays it instead of adding a second one.
#
# Usage, from anywhere, with the stack up:  sh docs/handbook/examples/seed-examples.sh
set -eu
here=$(cd "$(dirname "$0")" && pwd)
cd "$here/../../.."

dc() { MSYS_NO_PATHCONV=1 docker compose "$@"; }
if [ -f .env ]; then set -a; . ./.env; set +a; fi

A=00000000-0000-4000-8000-0000000000a1
CORE="http://127.0.0.1:${CORE_PORT:-18080}"
KC="http://127.0.0.1:${KEYCLOAK_PORT:-18081}"

# The admin's access token (client nava-web, password grant, as the Bruno collection does).
token=$(curl -fsS -X POST "$KC/realms/nava/protocol/openid-connect/token" \
  -d grant_type=password -d client_id=nava-web -d scope=openid \
  --data-urlencode "username=admin@tenant-a.test" \
  --data-urlencode "password=${ADMIN_A_PASSWORD:-dev-only-admin-a}" |
  sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
[ -n "$token" ] || { echo "seed-examples: no token for admin@tenant-a.test from $KC" >&2; exit 1; }

# api METHOD PATH [curl args...]: prints "<body>\n<http code>" so callers can branch on the code.
api() {
  m=$1; p=$2; shift 2
  curl -sS -X "$m" -H "Authorization: Bearer $token" -w '\n%{http_code}' "$@" "$CORE$p"
}
code_of() { tail -n 1; }
body_of() { sed '$d'; }

# Group ids of tenant A (rows written by group sync).
group_id() {
  dc exec -T postgres sh -c \
    "PGPASSWORD=\$(cat /run/pg_superuser_password) psql -qAt -v ON_ERROR_STOP=1 -U pgadmin -d nava" <<SQL | tr -d '\r'
SELECT id FROM groups WHERE tenant_id = '$A' AND path = '/t/$A/$1' AND deleted_at IS NULL;
SQL
}
admins=$(group_id admins); auditors=$(group_id auditors)
[ -n "$admins" ] && [ -n "$auditors" ] || { echo "seed-examples: no admins or auditors group in tenant A; run docs/acceptance/guide/tools/seed.sh first" >&2; exit 1; }

# publish_type KEY: create the type; when it exists, incident gets its extra field.
publish_type() {
  key=$1; f="$here/$key.json"
  out=$(api POST /api/v1/entity-types -H 'Content-Type: application/json' -H "Idempotency-Key: tick789-$key-create" --data-binary "@$f")
  case $(printf '%s\n' "$out" | code_of) in
    201) echo "seed-examples: created $key at version 1"; return ;;
    409) ;;
    *) echo "seed-examples: creating $key failed:" >&2; printf '%s\n' "$out" >&2; exit 1 ;;
  esac
  if api GET "/api/v1/entity-types/$key" | body_of | grep -q 'affected_service'; then
    echo "seed-examples: $key exists with the example definition"; return
  fi
  # The file's `definition` member is the body of a publish: {definition: ...}.
  body=$(sed '1,/^  "definition":/{/^  "definition":/!d;s/^  "definition": *//}' "$f" | sed '$d')
  out=$(api POST "/api/v1/entity-types/$key/versions" -H 'Content-Type: application/json' \
    -H "Idempotency-Key: tick789-$key-v2" --data-binary "{\"definition\": $body}")
  [ "$(printf '%s\n' "$out" | code_of)" = 201 ] || { echo "seed-examples: publishing $key v2 failed:" >&2; printf '%s\n' "$out" >&2; exit 1; }
  echo "seed-examples: $key exists; published the example definition as its next version"
}

# publish_access KEY: PUT the access policy once, with the group ids filled in.
publish_access() {
  key=$1
  cur=$(api GET "/api/v1/entity-types/$key/access")
  case $(printf '%s\n' "$cur" | code_of) in
    200) if printf '%s\n' "$cur" | body_of | grep -q '"auditors-read"'; then echo "seed-examples: $key access policy already published"; return; fi
         ver=$(printf '%s\n' "$cur" | body_of | sed -n 's/.*"version":\([0-9]*\).*/\1/p') ;;
    404) ver=0 ;;
    *) echo "seed-examples: reading $key access failed:" >&2; printf '%s\n' "$cur" >&2; exit 1 ;;
  esac
  policy=$(sed -e "s/__ADMINS__/$admins/g" -e "s/__AUDITORS__/$auditors/g" "$here/$key.access.json")
  out=$(api PUT "/api/v1/entity-types/$key/access" -H 'Content-Type: application/json' \
    -H "If-Match: W/\"$ver\"" -H "Idempotency-Key: tick789-$key-access-$ver" --data-binary "$policy")
  [ "$(printf '%s\n' "$out" | code_of)" = 200 ] || { echo "seed-examples: publishing $key access failed:" >&2; printf '%s\n' "$out" >&2; exit 1; }
  echo "seed-examples: published the $key access policy (masks included)"
}

# create_record KEY DATA: one record under a fixed Idempotency-Key.
create_record() {
  key=$1
  out=$(api POST "/api/v1/records/$key" -H 'Content-Type: application/json' -H "Idempotency-Key: tick789-$key-record" --data-binary "{\"data\": $2}")
  case $(printf '%s\n' "$out" | code_of) in
    201|200) printf 'seed-examples: record of %s: %s\n' "$key" "$(printf '%s\n' "$out" | body_of | sed -n 's/.*"number":"\([^"]*\)".*/\1/p')" ;;
    *) echo "seed-examples: creating a $key record failed:" >&2; printf '%s\n' "$out" >&2; exit 1 ;;
  esac
}

for k in incident loan_application visa_application; do publish_type "$k"; done
publish_access loan_application
publish_access visa_application

create_record incident '{"short_description":"Payments API returns 502","priority":2,"affected_service":"payments-api"}'
create_record loan_application '{"applicant_name":"Jordan Example","amount":250000,"term_months":240,"income":78000}'
create_record visa_application '{"applicant":"Sam Example","passport_number":"X1234567","visa_type":"student","travel_date":"2027-01-15"}'
echo "seed-examples: done. incident keeps the default ACL; loan_application and visa_application have their own."
