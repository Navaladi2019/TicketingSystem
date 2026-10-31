# Operator tenant onboarding

`POST /api/v1/operator/tenants` creates a tenant. It is served by its own process,
`nava-onboard`, with its own listener (`NAVA_ONBOARD_ADDR`, default `:8081`) and its
own admin listener. `nava-core`, which parses every tenant's input, never reads the
onboarding service account's secret or the `nava_onboard` database URL, and answers
the operator routes `501 not_implemented`.

The route takes an operator token only (realm role `nava-operator`, audience
`nava-operator-api`, and the `(iss, sub)` on `NAVA_OPERATORS`). A tenant token gets
403, a missing or invalid token 401.

## Request

Headers: `Authorization: Bearer <operator token>`, `Idempotency-Key: <1..255 chars>`.

```json
{
  "admin_email": "admin@acme.example",
  "email_domains": ["acme.example"],
  "identity_provider": {
    "provider_id": "oidc",
    "display_name": "Acme SSO",
    "config": {
      "clientId": "...", "clientSecret": "...",
      "authorizationUrl": "https://sso.acme.example/auth",
      "tokenUrl": "https://sso.acme.example/token",
      "jwksUrl": "https://sso.acme.example/jwks"
    }
  }
}
```

- `201` with `{"tenant_id", "state":"active"}` when the tenant was created, `200`
  for a replay of a completed request.
- The tenant id is `UUIDv5(onboarding namespace, operator principal + Idempotency-Key)`.
  A tenant id in the body, the query or `X-Tenant-Id` is ignored. Two operators
  using the same key get different tenants.
- Same key and same body resumes; same key and a different body is
  `409 idempotency_key_reuse` and creates nothing. The fingerprint is
  `idempotency.RequestHash` over the method, the path (not the query) and the body,
  stored in `tenants.onboarding_request_hash`. The `tenants` row in `provisioning`
  is the idempotency record; there is no `idempotency_keys` row.
- `409 domain_claimed`: another tenant owns one of the email domains.
- `409 onboarding_conflict`: the organization or IdP for this tenant, or the
  organization's binding, exists and is not this tenant's.
- `422 invalid_idp_config`: the IdP settings are not accepted (see below).

## Validation

- `email_domains`: 1 to 20, valid DNS names; **public email-provider domains are
  refused** (a short best-effort list, see AR-035). `admin_email` must be in one of
  them. Domain ownership is first-claim-wins: the operator calling the API attests
  the tenant owns the domains (AR-035).
- IdP `config` is an **allow-list of keys per provider** (the same list
  `CreateTenantOrg` enforces). An unknown key (`alias`, `kc.org.*`, SAML
  `artifactResolutionServiceUrl`, ...) is `422`.
- Every URL-valued key: https only; a host that is, or resolves to, a private,
  loopback, link-local, carrier-grade NAT, benchmark, reserved or multicast address,
  or a NAT64 address embedding one, is refused. The check happens once, at
  onboarding; Keycloak resolves the name again later, so DNS rebinding is an
  accepted residual (AR-035).
- `validateSignature=true` and `syncMode=IMPORT` are set by the platform whatever
  the request says. The IdP alias is derived from the tenant id.

## Steps and recovery

1. Per-tenant lock: a PostgreSQL advisory lock held in an open transaction for the
   whole call, so two calls for one tenant run one after the other. At most half the
   `nava_onboard` pool's connections run onboardings at once (each takes a lock
   connection and one for its statements).
2. `tenants` row in `provisioning` with the request hash, and one
   `tenant_email_domain` row per domain, in one transaction as `nava_onboard`
   (inside `InTenantTx(newTenantID)`). The domain claim is the ownership check:
   the primary key is on `domain` alone, so a domain another tenant owns is a
   unique violation on `tenant_email_domain_pkey`, mapped to `domain_claimed`
   (`409`). The transaction rolls back, so no `provisioning` row is left and
   Keycloak is never called. Two requests for one domain serialise on the key:
   one wins. A retry that finds its own `tenants` row inserts nothing (it
   already owns its domains). An `active` tenant ends here: a replay returns
   `200` and changes nothing. A request that fails after this step keeps its
   domains held by the `provisioning` tenant until the same key completes it.
3. Keycloak organization and IdP (`DirectoryAdmin.CreateTenantOrg`). A retry first
   asks `Directory.FindTenantOrg`: a complete org and IdP are reused, a lone
   organization is deleted and recreated, anything else is `onboarding_conflict`.
4. Binding, in one transaction as `nava_onboard`, while the tenant is still
   `provisioning`. `nava_onboard` has no SELECT on this table (its grants are the
   isolation control), so the retry decision is made without `ON CONFLICT`, which
   needs SELECT privilege and fails with 42501 under those grants. Under the
   per-tenant lock the service first reads, as `nava_app` with `WHERE tenant_id`
   (the clause scopes the read; no RLS policy does), the `(issuer, org id)` the
   tenant's binding row names. If it exists, an earlier call committed it. The
   same pair inserts nothing; a different pair (the organization was deleted
   outside the platform and this attempt resolved another) is
   `onboarding_conflict` and the tenant stays `provisioning`. If not, a plain
   INSERT runs; a unique violation on the `(issuer, org id)` key is
   `onboarding_conflict`. The service then reads ownership of every domain with
   `WHERE tenant_id` as a second check. Keycloak's refusal of a domain that is
   already linked to another organization (a `400` on organization create, step
   3) is also `domain_claimed`, as defence in depth.
5. `DirectoryAdmin.InviteUser` for the admin (see "Admin invite lifecycle"
   below). A refused invite (`ErrInviteRefused`)
   is answered exactly like a sent one. Any other invite error is `5xx`, on the
   first call and on every replay.
6. Activation (`tenants_activate`) is the last step.

A failure before step 6, including a failed invite, leaves the tenant `provisioning`;
the same key completes it. The tenant is never `active` without its binding, its
domains and an invited admin.

## Admin invite lifecycle (TICK-252)

`InviteUser` creates no Keycloak user for an email that has none. It writes a
`pending_invite` row (`tenant_id`, normalized `email`, `expires_at`,
`consumed_at`) as `nava_onboard`, only when the tenant owns the email's domain.
Re-onboarding moves the expiry of a pending invite and never revives a consumed
one. An email that already has a user takes the unchanged existing-user path
(`checkInvitable`).

1. The admin signs in through the tenant's IdP. `nava-first-broker-login`
   creates the user and adds them to the tenant's organization. Nothing is ever
   linked to an existing user, and `trustEmail` stays false. A user that already
   exists with that email makes this login fail closed.
2. The next group sync cycle (`nava-groupsync`, at most 5 minutes) places the
   user in `/t/<tenant_id>/` and consumes the invite, in one transaction with
   `app.tenant_id` set (`UPDATE ... WHERE consumed_at IS NULL AND expires_at > now()`).
   It does so only if the user is enabled, has an email equal to the invite's,
   the tenant's IdP asserted `email_verified=true` (see below),
   has at least one federated link and every link is an IdP bound to the tenant,
   is a member of the tenant's organization and of no other, and is in no
   tenant's group yet. Otherwise the invite stays as it was.
3. An invite not used within `NAVA_INVITE_TTL` expires. Run onboarding again with
   the same key and body to issue a new one, or invite again.

**The tenant's IdP must emit `email_verified`.** `CreateTenantOrg` gives every
tenant IdP one mapper that copies the upstream `email_verified` claim to the user
attribute `nava_idp_email_verified` (admin-only in realm `nava`'s user profile, so a
user cannot set it). Group sync places the invitee only when it is exactly `true`.
An IdP that sends no `email_verified`, or anything else, leaves the invited admin
unplaced and the invite pending: it fails closed. A SAML IdP has no standard
`email_verified` attribute, so it must send an attribute named `email_verified`
with the value `true`; the SAML mapper reads that attribute name. A tenant user's
email is admin-only in realm `nava`'s user profile, so it comes from the IdP and
nowhere else. Microsoft Entra does not emit a
usable `email_verified` by default; map the verified-email claim (`xms_edov`) or a
verified-domain claim to `email_verified` before onboarding a tenant on Entra. An
IdP bound to a multi-tenant issuer (such as Entra `common`) or with open
self-registration and no email verification must not be used for a tenant.

The pending admin cannot sign in before step 1: there is no local user and no
password. The accepted squatting risk is AR-037.

## Configuration (nava-onboard)

On the compose stack `nava-onboard` is its own service on the `internal` network,
with no published port: the tenant ingress (nava-core's port) answers the operator
route 501. From the host, call it inside its container, as `make stack-test-onboard`
does, with the operator token in the environment, not on the command line:

```sh
NAVA_OPERATOR_TOKEN=$(<command that prints the operator access token>) \
docker compose exec -T -e NAVA_OPERATOR_TOKEN nava-onboard sh -c \
  'exec wget -S -O - --header "Authorization: Bearer $NAVA_OPERATOR_TOKEN" \
     --header "Idempotency-Key: $1" --header "Content-Type: application/json" \
     --post-data "$2" http://127.0.0.1:8081/api/v1/operator/tenants' \
  sh '<idempotency key>' '<body>'
```

`-e NAVA_OPERATOR_TOKEN` with no value copies the variable from your shell into
the container, and the single quotes keep your shell from expanding it, so the
token never appears in an argument list.

Get the operator token from Keycloak's published port (`http://localhost:18081`,
or your `KEYCLOAK_PUBLIC_URL`). Its issuer is the pinned one (`KC_HOSTNAME`), whatever
Host the request used, and `NAVA_OIDC_ISSUERS`, `NAVA_ONBOARD_ISSUER` and the issuer
in `NAVA_OPERATORS` must all be exactly that value, the one `nava-core` trusts: the
binding issuer is what `GET /api/v1/me` resolves the new admin's tenant by. On the
compose stack they are set to `${KEYCLOAK_PUBLIC_URL:-http://localhost:18081}/realms/nava`,
and `NAVA_OIDC_FETCH_BASE_URL` (`http://keycloak:8080`) is where keys are fetched.

The operator token is a short-lived access token. Never paste it inline into a
command: argv is visible in the process list on the host and in the container,
and lands in shell history. Not `nc`: it half-closes the connection
when its input ends, and the server cancels the request on that EOF. The admin's first sign-in resolves in
`GET /api/v1/me` only after a `nava-groupsync` cycle has synced them.


`NAVA_ONBOARD_ADDR`, `NAVA_ADMIN_ADDR`, `NAVA_DATABASE_URL` (as `nava_app`),
`NAVA_ONBOARD_DATABASE_URL` (as `nava_onboard`), `NAVA_OIDC_ISSUERS`,
`NAVA_ONBOARD_ISSUER` (one of them), `NAVA_OPERATORS` (`issuer|subject,...`),
`NAVA_OIDC_FETCH_BASE_URL` (optional back channel for discovery and JWKS),
`NAVA_KEYCLOAK_URL`, `NAVA_KEYCLOAK_REALM`, `NAVA_DIRECTORY_CLIENT_ID/SECRET` (runtime
account, for the invite) and `NAVA_ONBOARD_CLIENT_ID/SECRET` (onboarding account).
`NAVA_INVITE_TTL` (default `168h`, at most `720h`) is how long a pending admin invite
stays valid. Every one but the realm, the TTL and the fetch base URL is required; the process refuses to start without them.

## Retry fails with admin group missing

**Detect.** Onboarding stops at the admins-group step, the tenant stays
`provisioning`, and every retry with the same key fails. `nava-onboard` logs one
ERROR line each time:

```json
{"level":"ERROR","msg":"onboard: admins group has no groups row","tenant_id":"<tenant uuid>","keycloak_group_id":"<current Keycloak group id>"}
```

The error is `ErrAdminGroupMissing` (TICK-369). Cause: the `admins` group under
`/t/<tenant_id>/` was deleted and re-created in Keycloak, so `groups` still holds a
stale row for it under the old Keycloak id. Group sync does not repair this on its
own (it would re-point which Keycloak group grants tenant admin; D-P10). The fix
is this operator step.

**Confirm.** Run the queries as the bootstrap superuser, `pgadmin`, over the
Unix socket: it is the only role that can. `groups`, `tenant_admin_group` and
`tenants` are under `FORCE ROW LEVEL SECURITY`; the migration owner
(`nava_migrator`) and every runtime role are `NOBYPASSRLS`, so without
`app.tenant_id` they see no rows and the repair would change nothing (`UPDATE 0`).
`pg_hba.conf` lets `pgadmin` in over the socket only, which only the `postgres`
container mounts. On the compose stack, from the repository directory, with the
two ids from the log line:

```sh
docker compose exec -it postgres sh -c \
  'export PGPASSWORD="$(cat /run/pg_superuser_password)"; \
   exec psql -h /var/run/postgresql -U pgadmin -d nava "$@"' \
  sh -v ON_ERROR_STOP=1 \
     -v tenant_id=<tenant uuid> -v kc_group_id=<keycloak group id>
```

The password is read inside the container and never appears in an argument list.
Set `ON_ERROR_STOP=1` first, so psql stops at a failed statement, and run the
blocks below one at a time.
`:'tenant_id'` and `:'kc_group_id'` below are those psql variables.

```sql
-- The stale row: the tenant's admins group, under an id Keycloak no longer has.
SELECT id, keycloak_group_id, path
  FROM groups
 WHERE tenant_id = :'tenant_id'::uuid
   AND path = '/t/' || :'tenant_id' || '/admins';

-- The tenant must have no tenant_admin_group row yet (0 rows), and the current
-- Keycloak id must have no groups row (0 rows), or this is a different problem.
SELECT * FROM tenant_admin_group WHERE tenant_id = :'tenant_id'::uuid;
SELECT * FROM groups WHERE tenant_id = :'tenant_id'::uuid AND keycloak_group_id = :'kc_group_id';
```

Expect one stale row whose `keycloak_group_id` differs from the logged one. If the
first query returned more than one row, or the other two returned any, stop: the
cause is not this one. Three empty results mean you are not connected as
`pgadmin`.

**Check Keycloak.** Before any repair, check that the logged `keycloak_group_id`
is the tenant's admins group in Keycloak; the repair trusts it. From the repository
directory, with the id from the log line. It must print `"path" : "/t/<tenant
uuid>/admins"` and exit 0; a non-zero exit (an unknown id, a failed login) or any
other path means stop. The admin password reaches `kcadm.sh` through
`KC_CLI_PASSWORD`, not `--password`, so it is in no process argument list, and the
temporary config file, which holds a master-realm refresh token, is removed on exit.

```sh
docker compose exec keycloak sh -c \
  'cd /opt/keycloak/bin && KCADM_CONFIG=$(mktemp) || exit 1
   trap "rm -f \"\$KCADM_CONFIG\"" EXIT; trap "exit 1" HUP INT TERM
   KC_CLI_PASSWORD="$KC_BOOTSTRAP_ADMIN_PASSWORD" ./kcadm.sh config credentials --config "$KCADM_CONFIG" \
     --server http://localhost:8080 --realm master --user "$KC_BOOTSTRAP_ADMIN_USERNAME" >/dev/null &&
   ./kcadm.sh get groups/"$1" --config "$KCADM_CONFIG" -r nava --fields id,path' \
  sh <keycloak group id>
```

**Repair.** Run one option's block, which opens a transaction and runs one
statement. Read its tag before you commit: it must be exactly `UPDATE 1` or
`DELETE 1`. Then run the `COMMIT` block; for any other tag, run the `ROLLBACK`
block and stop.

Option A (preferred): point the stale row at the current Keycloak id. Its platform
id and memberships stay; the next group sync cycle (at most 5 minutes) refreshes
the memberships.

```sql
BEGIN;
UPDATE groups
   SET keycloak_group_id = :'kc_group_id'
 WHERE tenant_id = :'tenant_id'::uuid
   AND path = '/t/' || :'tenant_id' || '/admins';
```

Option B: remove the stale row. Its `group_members` and any `tenant_admin_group`
row go with it (ON DELETE CASCADE); retry inserts a fresh row, and group sync
re-adds the members. `path` is not unique, so check that the tag is `DELETE 1`.

```sql
BEGIN;
DELETE FROM groups
 WHERE tenant_id = :'tenant_id'::uuid
   AND path = '/t/' || :'tenant_id' || '/admins';
```

The tag was exactly 1:

```sql
COMMIT;
```

Any other tag:

```sql
ROLLBACK;
```

**Retry.** Run the original onboarding request again with the same
`Idempotency-Key` and body (see "Steps and recovery"). The admins-group step now
finds a row for the current id, writes `tenant_admin_group`, and onboarding
continues to the invite and activation. Check:

```sql
SELECT t.state, a.group_id, g.keycloak_group_id
  FROM tenants t
  JOIN tenant_admin_group a ON a.tenant_id = t.id
  JOIN groups g ON (g.tenant_id, g.id) = (a.tenant_id, a.group_id)
 WHERE t.id = :'tenant_id'::uuid;   -- state active, the current Keycloak id
```

A second occurrence, or any automation that re-creates Keycloak groups, goes to the
Architecture squad as an RFC for automated repair (TICK-372).

## TODO: abandoning a provisioning tenant

There is no way to abandon a `provisioning` tenant. A tenant whose onboarding never
completes keeps its Keycloak organization, its IdP, and (once bound) its
`tenant_identity_binding` and `tenant_email_domain` rows, so the domains stay
claimed. Only the operator's own retry (same key and body) can finish or reuse it.
Recorded as AR-036, which also covers the tenant a binding/org mismatch
(`onboarding_conflict` on retry) leaves `provisioning`; needs an operator
"abandon" operation that removes the Keycloak
objects and releases the rows.
