#!/bin/sh
# Acceptance guide (TICK-287): seeds the two demo tenants of realm `nava` into
# the platform database and runs one group sync cycle, so the seeded Keycloak
# users (alice, admin-a, bob) resolve in GET /api/v1/me.
#
# S8 (TICK-346) adds what onboarding gives every new tenant, which these two
# tenants never went through: tenant A's `admins` group (Keycloak group
# /t/<A>/admins with admin@tenant-a.test in it, its `groups` row from group
# sync, and the `tenant_admin_group` row; TODO-S8-01), and the `incident`,
# `request` and `change` types at version 1 in both tenants, from
# internal/metadata/templates/. The version 1 schema is the template's
# `schema` member as written in the file (json `->` keeps its bytes), so its
# schema_sha256 is the SHA-256 of those bytes, as onboarding writes it.
#
# The stack has no seed command or onboarding of the realm's fixed tenants, so
# this writes what S3 and S5's acceptance kits wrote by hand: tenants A and B
# (`active`), their identity binding under the pinned issuer, and their email
# domains, as the bootstrap superuser over the postgres container's Unix
# socket. Idempotent: run it again at any time.
#
# Usage, from anywhere, with the stack up:  sh docs/acceptance/guide/tools/seed.sh
set -eu
cd "$(dirname "$0")/../../../.."

# Git Bash would rewrite /run/... in the container command.
MSYS_NO_PATHCONV=1
export MSYS_NO_PATHCONV

# .env applies as it does to compose (KEYCLOAK_PUBLIC_URL, COMPOSE_PROJECT_NAME).
if [ -f .env ]; then set -a; . ./.env; set +a; fi

ISSUER="${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}/realms/nava"
A=00000000-0000-4000-8000-0000000000a1
B=00000000-0000-4000-8000-0000000000b1

echo "seed: tenants A and B, issuer $ISSUER"
docker compose exec -T postgres sh -c \
  'PGPASSWORD=$(cat /run/pg_superuser_password) psql -q -v ON_ERROR_STOP=1 -U pgadmin -d nava' <<SQL
INSERT INTO tenants (id, state) VALUES ('$A', 'active'), ('$B', 'active')
  ON CONFLICT (id) DO NOTHING;
INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES
  ('$A', '$ISSUER', '$A', '{idp-tenant-a}'),
  ('$B', '$ISSUER', '$B', '{idp-tenant-b}')
  ON CONFLICT DO NOTHING;
INSERT INTO tenant_email_domain (tenant_id, domain) VALUES
  ('$A', 'tenant-a.test'), ('$B', 'tenant-b.test')
  ON CONFLICT DO NOTHING;
SELECT t.id AS tenant, t.state, b.issuer FROM tenants t JOIN tenant_identity_binding b ON b.tenant_id = t.id
 WHERE t.id IN ('$A', '$B') ORDER BY 1;
SQL

# Tenant A's admins group in Keycloak, with admin@tenant-a.test in it
# (onboarding creates it for a new tenant). Idempotent: 409 means it exists.
KC="http://127.0.0.1:${KEYCLOAK_PORT:-18081}"
kc_token=$(curl -fsS -X POST "$KC/realms/master/protocol/openid-connect/token" \
  -d grant_type=password -d client_id=admin-cli \
  --data-urlencode "username=${KEYCLOAK_ADMIN:-admin}" \
  --data-urlencode "password=${KEYCLOAK_ADMIN_PASSWORD:-dev-only-admin}" |
  sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
[ -n "$kc_token" ] || { echo "seed: no Keycloak admin token from $KC" >&2; exit 1; }
kc() { curl -fsS -H "Authorization: Bearer $kc_token" -H 'Content-Type: application/json' "$@"; }
first_id() { sed -n 's/^[^{]*{"id":"\([^"]*\)".*/\1/p' | head -n 1; }
parent=$(kc "$KC/admin/realms/nava/group-by-path/t/$A" | first_id)
[ -n "$parent" ] || { echo "seed: no group /t/$A in realm nava" >&2; exit 1; }
code=$(curl -sS -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $kc_token" \
  -H 'Content-Type: application/json' -d '{"name":"admins"}' "$KC/admin/realms/nava/groups/$parent/children")
case $code in 201|409) ;; *) echo "seed: creating /t/$A/admins answered HTTP $code" >&2; exit 1 ;; esac
admins=$(kc "$KC/admin/realms/nava/group-by-path/t/$A/admins" | first_id)
user=$(kc "$KC/admin/realms/nava/users?username=admin%40tenant-a.test&exact=true" | sed 's/^\[//' | first_id)
[ -n "$admins" ] && [ -n "$user" ] || { echo "seed: no admins group or no admin@tenant-a.test" >&2; exit 1; }
kc -X PUT "$KC/admin/realms/nava/users/$user/groups/$admins" >/dev/null
echo "seed: admin@tenant-a.test is in /t/$A/admins"

echo "seed: one group sync cycle (nava-groupsync --once)"
docker compose run --rm --no-deps nava-groupsync --once

# The admins group's platform id, the entity types at version 1. Template
# bytes go in as dollar-quoted literals; the files hold no "$tpl$".
tpl() { cat "internal/metadata/templates/$1.json"; }
echo "seed: tenant A's admin group, incident/request/change at version 1 in A and B"
docker compose exec -T postgres sh -c \
  'PGPASSWORD=$(cat /run/pg_superuser_password) psql -q -v ON_ERROR_STOP=1 -U pgadmin -d nava' <<SQL
INSERT INTO tenant_admin_group (tenant_id, group_id)
  SELECT tenant_id, id FROM groups WHERE tenant_id = '$A' AND path = '/t/$A/admins'
  ON CONFLICT DO NOTHING;
DO \$\$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM tenant_admin_group WHERE tenant_id = '$A') THEN
    RAISE EXCEPTION 'no groups row for /t/$A/admins after group sync';
  END IF;
END \$\$;
CREATE TEMP TABLE seed_tpl (name text, doc json);
INSERT INTO seed_tpl VALUES
  ('Incident', \$tpl\$$(tpl incident)\$tpl\$),
  ('Request', \$tpl\$$(tpl request)\$tpl\$),
  ('Change', \$tpl\$$(tpl change)\$tpl\$);
INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
  SELECT t.id, gen_random_uuid(), s.doc->>'key', s.name, s.doc->>'number_prefix', 1
  FROM seed_tpl s CROSS JOIN tenants t WHERE t.id IN ('$A', '$B')
  ON CONFLICT DO NOTHING;
INSERT INTO entity_definition_versions
  (tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
  SELECT d.tenant_id, gen_random_uuid(), d.id, 1, s.doc->'schema',
         sha256(convert_to((s.doc->'schema')::text, 'UTF8')),
         (s.doc->'fixed_columns')::jsonb, (s.doc->'states')::jsonb, 'initial', NULL
  FROM seed_tpl s JOIN entity_definitions d ON d.key = s.doc->>'key'
  WHERE d.tenant_id IN ('$A', '$B')
  ON CONFLICT DO NOTHING;
SELECT d.tenant_id AS tenant, d.key, d.current_version, encode(v.schema_sha256, 'hex') AS v1_schema_sha256
  FROM entity_definitions d JOIN entity_definition_versions v
    ON v.tenant_id = d.tenant_id AND v.definition_id = d.id AND v.version = 1
 WHERE d.tenant_id IN ('$A', '$B') ORDER BY 1, 2;
SQL
echo "seed: done. alice@tenant-a.test, admin@tenant-a.test and bob@tenant-b.test now resolve in GET /api/v1/me;"
echo "seed: admin@tenant-a.test is tenant A's admin; both tenants have incident, request and change."
