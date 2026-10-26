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
  refused** (a short best-effort list, see AR-031). `admin_email` must be in one of
  them. Domain ownership is first-claim-wins: the operator calling the API attests
  the tenant owns the domains (AR-031).
- IdP `config` is an **allow-list of keys per provider** (the same list
  `CreateTenantOrg` enforces). An unknown key (`alias`, `kc.org.*`, SAML
  `artifactResolutionServiceUrl`, ...) is `422`.
- Every URL-valued key: https only; a host that is, or resolves to, a private,
  loopback, link-local, carrier-grade NAT, benchmark, reserved or multicast address,
  or a NAT64 address embedding one, is refused. The check happens once, at
  onboarding; Keycloak resolves the name again later, so DNS rebinding is an
  accepted residual (AR-031).
- `validateSignature=true` and `syncMode=IMPORT` are set by the platform whatever
  the request says. The IdP alias is derived from the tenant id.

## Steps and recovery

1. Per-tenant lock: a PostgreSQL advisory lock held in an open transaction for the
   whole call, so two calls for one tenant run one after the other. At most half the
   `nava_onboard` pool's connections run onboardings at once (each takes a lock
   connection and one for its statements).
2. `tenants` row in `provisioning` with the request hash (as `nava_onboard`, inside
   `InTenantTx(newTenantID)`). An `active` tenant ends here: a replay returns `200`
   and changes nothing.
3. Keycloak organization and IdP (`DirectoryAdmin.CreateTenantOrg`). A retry first
   asks `Directory.FindTenantOrg`: a complete org and IdP are reused, a lone
   organization is deleted and recreated, anything else is `onboarding_conflict`.
4. Binding and one `tenant_email_domain` row per domain, in one transaction, while
   the tenant is still `provisioning`. The inserts are `ON CONFLICT ON CONSTRAINT
   <primary key> DO NOTHING`, so a retry skips the tenant's own rows. A skipped
   domain row may belong to another tenant, so the service then reads ownership of
   every domain under the tenant predicate and answers `domain_claimed` if any is
   not the tenant's.
5. `DirectoryAdmin.InviteUser` for the admin. A refused invite (`ErrInviteRefused`)
   is answered exactly like a sent one. Any other invite error is `5xx`, on the
   first call and on every replay.
6. Activation (`tenants_activate`) is the last step.

A failure before step 6, including a failed invite, leaves the tenant `provisioning`;
the same key completes it. The tenant is never `active` without its binding, its
domains and an invited admin.

## Configuration (nava-onboard)

`NAVA_ONBOARD_ADDR`, `NAVA_ADMIN_ADDR`, `NAVA_DATABASE_URL` (as `nava_app`),
`NAVA_ONBOARD_DATABASE_URL` (as `nava_onboard`), `NAVA_OIDC_ISSUERS`,
`NAVA_ONBOARD_ISSUER` (one of them), `NAVA_OPERATORS` (`issuer|subject,...`),
`NAVA_KEYCLOAK_URL`, `NAVA_KEYCLOAK_REALM`, `NAVA_DIRECTORY_CLIENT_ID/SECRET` (runtime
account, for the invite) and `NAVA_ONBOARD_CLIENT_ID/SECRET` (onboarding account).
Every one but the realm is required; the process refuses to start without them.
