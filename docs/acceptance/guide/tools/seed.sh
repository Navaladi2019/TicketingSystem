#!/bin/sh
# Acceptance guide (TICK-287): seeds the two demo tenants of realm `nava` into
# the platform database and runs one group sync cycle, so the seeded Keycloak
# users (alice, admin-a, bob) resolve in GET /api/v1/me.
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

echo "seed: one group sync cycle (nava-groupsync --once)"
docker compose run --rm --no-deps nava-groupsync --once
echo "seed: done. alice@tenant-a.test, admin@tenant-a.test and bob@tenant-b.test now resolve in GET /api/v1/me."
