# Operator tenant onboarding

`POST /api/v1/operator/tenants` creates a tenant. It takes an operator token only
(realm role `nava-operator`, audience `nava-operator-api`, and the `(iss, sub)`
on `NAVA_OPERATORS`). A tenant token gets 403, a missing or invalid token 401.

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
- `domain_claimed` (409): another tenant owns one of the email domains.
- `onboarding_conflict` (409): the organization or IdP for this tenant exists and
  is not a complete, policy-conforming pair.

## Validation

- `email_domains`: 1 to 20, valid DNS names, **public email-provider domains are
  refused** (for example `gmail.com`). `admin_email` must be in one of them.
  Domain ownership is first-claim-wins: the operator calling the API attests the
  tenant owns the domains (AR-031).
- IdP URLs (`authorizationUrl`, `tokenUrl`, `jwksUrl`, `userInfoUrl`, `logoutUrl`,
  `issuer`, `metadataDescriptorUrl`, `singleSignOnServiceUrl`,
  `singleLogoutServiceUrl`): https only; a host that is, or resolves to, a private,
  loopback, link-local, carrier-grade NAT or multicast address is refused.
- `validateSignature=true` and `syncMode=IMPORT` are set by the platform whatever
  the request says. The IdP alias is derived from the tenant id; a request field
  naming one is ignored.

## Steps and recovery

1. Per-tenant lock (a PostgreSQL advisory lock held in an open transaction for the
   whole call, so two concurrent calls for one tenant run one after the other).
2. `tenants` row in `provisioning` with the request hash (as `nava_onboard`,
   inside `InTenantTx(newTenantID)`).
3. Keycloak organization and IdP (`DirectoryAdmin.CreateTenantOrg`). A retry first
   asks `Directory.FindTenantOrg`: a complete org and IdP are reused, a lone
   organization is deleted and recreated.
4. In one transaction: `tenant_identity_binding`, one `tenant_email_domain` row per
   domain, and `tenants.state = 'active'`.
5. `DirectoryAdmin.InviteUser` for the admin. A refused invite (`ErrInviteRefused`)
   is answered exactly like a sent one. If the invite call fails the tenant is
   already `active`; the call returns 500 and a retry with the same key repeats the
   invite.

A failure before step 4 leaves the tenant `provisioning`; the same key completes it.

## Configuration

`NAVA_OPERATORS` (`issuer|subject,...`), `NAVA_ONBOARD_DATABASE_URL` (as
`nava_onboard`), `NAVA_KEYCLOAK_URL`, `NAVA_KEYCLOAK_REALM`, `NAVA_ONBOARD_ISSUER`
(one of `NAVA_OIDC_ISSUERS`), `NAVA_DIRECTORY_CLIENT_ID/SECRET` (runtime account) and
`NAVA_ONBOARD_CLIENT_ID/SECRET` (onboarding account). With `NAVA_ONBOARD_DATABASE_URL`
unset the route answers 501.
