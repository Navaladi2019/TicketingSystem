# S4 acceptance evidence (TICK-46)

Recorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on 2026-10-28T21:46:03Z. Tokens are never recorded; the IdP client secret is redacted.

Tenant A: `0d216d6e-6961-5636-8a07-8b61059b813c`, domain `onb-0d216d6e.test`, admin `admin@onb-0d216d6e.test`.
Tenant B: `de775309-b5ba-517f-a6dd-29f1c71a299d`, domain `onb-de775309.test`, admin `admin@onb-de775309.test`.


## Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing

First call, tenant onb-0d216d6e.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: a9d25a15-ad8b-4db7-8786-aad89ec454ae
Content-Type: application/json

{
  "admin_email": "admin@onb-0d216d6e.test",
  "email_domains": [
    "onb-0d216d6e.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Onboarded onb-0d216d6e.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"0d216d6e-6961-5636-8a07-8b61059b813c"}
```

- **PASS** first call: 201, tenant_id `0d216d6e-6961-5636-8a07-8b61059b813c`, state `active` (want 201, `0d216d6e-6961-5636-8a07-8b61059b813c`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: a9d25a15-ad8b-4db7-8786-aad89ec454ae
Content-Type: application/json

{
  "admin_email": "admin@onb-0d216d6e.test",
  "email_domains": [
    "onb-0d216d6e.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Onboarded onb-0d216d6e.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"0d216d6e-6961-5636-8a07-8b61059b813c"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `3 tenants, 3 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-0d216d6e-6961-5636-8a07-8b61059b813c`, after `3 tenants, 3 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-0d216d6e-6961-5636-8a07-8b61059b813c`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: a9d25a15-ad8b-4db7-8786-aad89ec454ae
Content-Type: application/json

{
  "admin_email": "admin@onb-0d216d6e.test",
  "email_domains": [
    "onb-0d216d6e.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Changed onb-0d216d6e.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

(no body: busybox wget keeps none for an error status)
```

- **PASS** same key with a different body: 409 (want 409 `idempotency_key_reuse`)
- **PASS** nothing new after the conflicting call
- **PASS** the admin `admin@onb-0d216d6e.test` has a pending invite and no Keycloak user yet
First call, tenant onb-de775309.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 5ffbf7bb-f74e-4cfd-b2a9-b833562bcb5c
Content-Type: application/json

{
  "admin_email": "admin@onb-de775309.test",
  "email_domains": [
    "onb-de775309.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Onboarded onb-de775309.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"de775309-b5ba-517f-a6dd-29f1c71a299d"}
```

- **PASS** first call: 201, tenant_id `de775309-b5ba-517f-a6dd-29f1c71a299d`, state `active` (want 201, `de775309-b5ba-517f-a6dd-29f1c71a299d`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 5ffbf7bb-f74e-4cfd-b2a9-b833562bcb5c
Content-Type: application/json

{
  "admin_email": "admin@onb-de775309.test",
  "email_domains": [
    "onb-de775309.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Onboarded onb-de775309.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"de775309-b5ba-517f-a6dd-29f1c71a299d"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `4 tenants, 4 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-de775309-b5ba-517f-a6dd-29f1c71a299d`, after `4 tenants, 4 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-de775309-b5ba-517f-a6dd-29f1c71a299d`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 5ffbf7bb-f74e-4cfd-b2a9-b833562bcb5c
Content-Type: application/json

{
  "admin_email": "admin@onb-de775309.test",
  "email_domains": [
    "onb-de775309.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Changed onb-de775309.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

(no body: busybox wget keeps none for an error status)
```

- **PASS** same key with a different body: 409 (want 409 `idempotency_key_reuse`)
- **PASS** nothing new after the conflicting call
- **PASS** the admin `admin@onb-de775309.test` has a pending invite and no Keycloak user yet

## Criterion 2: the new admin signs in and is scoped to the new tenant

- **PASS** `admin@onb-0d216d6e.test` signed in through IdP `idp-0d216d6e-6961-5636-8a07-8b61059b813c`; the token's `organization` claim is exactly `t-0d216d6e-6961-5636-8a07-8b61059b813c`
- **PASS** `admin@onb-de775309.test` signed in through IdP `idp-de775309-b5ba-517f-a6dd-29f1c71a299d`; the token's `organization` claim is exactly `t-de775309-b5ba-517f-a6dd-29f1c71a299d`
One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).

Admin of onb-0d216d6e.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-0d216d6e.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-0d216d6e-6961-5636-8a07-8b61059b813c","tenant_id":"0d216d6e-6961-5636-8a07-8b61059b813c","user_id":"fa5086f8-08a5-49b7-9150-697ca695a741"}
```

- **PASS** `/me` is tenant `0d216d6e-6961-5636-8a07-8b61059b813c` via `idp-0d216d6e-6961-5636-8a07-8b61059b813c` (got 200, tenant `0d216d6e-6961-5636-8a07-8b61059b813c`, idp `idp-0d216d6e-6961-5636-8a07-8b61059b813c`)
- **PASS** `admin@onb-0d216d6e.test` is in `/t/0d216d6e-6961-5636-8a07-8b61059b813c/` and the invite is consumed
Admin of onb-de775309.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-de775309.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-de775309-b5ba-517f-a6dd-29f1c71a299d","tenant_id":"de775309-b5ba-517f-a6dd-29f1c71a299d","user_id":"32734f64-2d86-4b60-8d08-30b54139c001"}
```

- **PASS** `/me` is tenant `de775309-b5ba-517f-a6dd-29f1c71a299d` via `idp-de775309-b5ba-517f-a6dd-29f1c71a299d` (got 200, tenant `de775309-b5ba-517f-a6dd-29f1c71a299d`, idp `idp-de775309-b5ba-517f-a6dd-29f1c71a299d`)
- **PASS** `admin@onb-de775309.test` is in `/t/de775309-b5ba-517f-a6dd-29f1c71a299d/` and the invite is consumed

## Isolation check: tenant B cannot see or change what onboarding created for tenant A

Admin of onb-de775309.test names tenant onb-0d216d6e.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-de775309.test token>
X-Tenant-Id: 0d216d6e-6961-5636-8a07-8b61059b813c
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-de775309-b5ba-517f-a6dd-29f1c71a299d","tenant_id":"de775309-b5ba-517f-a6dd-29f1c71a299d","user_id":"32734f64-2d86-4b60-8d08-30b54139c001"}
```

- **PASS** still tenant `de775309-b5ba-517f-a6dd-29f1c71a299d`, never `0d216d6e-6961-5636-8a07-8b61059b813c`
Admin of onb-de775309.test names tenant onb-0d216d6e.test:

```http
GET /api/v1/me?tenant_id=0d216d6e-6961-5636-8a07-8b61059b813c (nava-core)
Authorization: Bearer <admin@onb-de775309.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-de775309-b5ba-517f-a6dd-29f1c71a299d","tenant_id":"de775309-b5ba-517f-a6dd-29f1c71a299d","user_id":"32734f64-2d86-4b60-8d08-30b54139c001"}
```

- **PASS** still tenant `de775309-b5ba-517f-a6dd-29f1c71a299d`, never `0d216d6e-6961-5636-8a07-8b61059b813c`
- **PASS** `admin@onb-de775309.test` is not in `/t/0d216d6e-6961-5636-8a07-8b61059b813c/` and not a member of organization `t-0d216d6e-6961-5636-8a07-8b61059b813c`
- **PASS** `admin@onb-de775309.test` holds no link to `idp-0d216d6e-6961-5636-8a07-8b61059b813c`
Admin of onb-0d216d6e.test names tenant onb-de775309.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-0d216d6e.test token>
X-Tenant-Id: de775309-b5ba-517f-a6dd-29f1c71a299d
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-0d216d6e-6961-5636-8a07-8b61059b813c","tenant_id":"0d216d6e-6961-5636-8a07-8b61059b813c","user_id":"fa5086f8-08a5-49b7-9150-697ca695a741"}
```

- **PASS** still tenant `0d216d6e-6961-5636-8a07-8b61059b813c`, never `de775309-b5ba-517f-a6dd-29f1c71a299d`
Admin of onb-0d216d6e.test names tenant onb-de775309.test:

```http
GET /api/v1/me?tenant_id=de775309-b5ba-517f-a6dd-29f1c71a299d (nava-core)
Authorization: Bearer <admin@onb-0d216d6e.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-0d216d6e-6961-5636-8a07-8b61059b813c","tenant_id":"0d216d6e-6961-5636-8a07-8b61059b813c","user_id":"fa5086f8-08a5-49b7-9150-697ca695a741"}
```

- **PASS** still tenant `0d216d6e-6961-5636-8a07-8b61059b813c`, never `de775309-b5ba-517f-a6dd-29f1c71a299d`
- **PASS** `admin@onb-0d216d6e.test` is not in `/t/de775309-b5ba-517f-a6dd-29f1c71a299d/` and not a member of organization `t-de775309-b5ba-517f-a6dd-29f1c71a299d`
- **PASS** `admin@onb-0d216d6e.test` holds no link to `idp-de775309-b5ba-517f-a6dd-29f1c71a299d`
A new tenant claims tenant A's domain:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 828b3f6d-36c9-4c63-9c52-26772dfee1b8
Content-Type: application/json

{
  "admin_email": "admin@onb-0d216d6e.test",
  "email_domains": [
    "onb-0d216d6e.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Onboarded onb-0d216d6e.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

{"type":"urn:nava:problem:domain_claimed","title":"Email domain already claimed","status":409,"code":"domain_claimed","detail":"An email domain of the request is already claimed by a tenant.","instance":"urn:nava:request:2346905a80c8e84e3f0cc84e75549ce4"}
```

- **PASS** 409 `domain_claimed` (want 409 `domain_claimed`)
- **PASS** the body names neither tenant A (`0d216d6e-6961-5636-8a07-8b61059b813c`) nor its organization (`t-0d216d6e-6961-5636-8a07-8b61059b813c`, id `e4d13aa0-a000-4b68-9ca8-9b7b931d8ad4`) nor its IdP
- **PASS** `onb-0d216d6e.test` is still owned by tenant A (got `0d216d6e-6961-5636-8a07-8b61059b813c`)
- **PASS** no `tenants` row (provisioning or otherwise) for the refused tenant `e0045390-e5f0-5c13-84d2-2ab9987bce61` (got 0)
- **PASS** no Keycloak organization `t-e0045390-e5f0-5c13-84d2-2ab9987bce61` (got [])
- **PASS** no `tenant_email_domain` row for the refused tenant (got 0)
- **PASS** no second invite for `admin@onb-0d216d6e.test`
Tenant B's admin replays A's onboarding on nava-core:

```http
POST /api/v1/operator/tenants (nava-core)
Authorization: Bearer <admin@onb-de775309.test token>
Idempotency-Key: a9d25a15-ad8b-4db7-8786-aad89ec454ae
```

```http
HTTP/1.1 501

{"type":"urn:nava:problem:not_implemented","title":"Not implemented","status":501,"code":"not_implemented","detail":"This operation is declared in the contract but not implemented yet.","instance":"urn:nava:request:64c767d495f9cf671455e4c58944d388"}
```

- **PASS** refused with 501; nothing changes for tenant A (`4 tenants, 4 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-0d216d6e-6961-5636-8a07-8b61059b813c`)

## Criterion 3: inviting a user already federated to another tenant's IdP is rejected

- **PASS** precondition: `lead-bafd0f2c@onb-d8bdba37.test` exists and is federated to tenant B's IdP only (`[idp-de775309-b5ba-517f-a6dd-29f1c71a299d]`)
Onboard tenant C naming that user as admin:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e9d2d188-ac78-40a8-829b-8509496cf98c
Content-Type: application/json

{
  "admin_email": "lead-bafd0f2c@onb-d8bdba37.test",
  "email_domains": [
    "onb-d8bdba37.test"
  ],
  "identity_provider": {
    "config": {
      "authorizationUrl": "https://93.184.216.34/auth",
      "clientAuthMethod": "client_secret_post",
      "clientId": "nava-broker",
      "clientSecret": "\u003credacted\u003e",
      "defaultScope": "openid email profile",
      "jwksUrl": "https://93.184.216.34/jwks",
      "tokenUrl": "https://93.184.216.34/token",
      "useJwksUrl": "true"
    },
    "display_name": "Onboarded onb-d8bdba37.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"d8bdba37-661f-5ee8-9dab-3dddfe18a269"}
```

Status 201. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.

- **PASS** no invite for `lead-bafd0f2c@onb-d8bdba37.test` in tenant C (got `missing`)
- **PASS** signing in through tenant C's IdP as `lead-bafd0f2c@onb-d8bdba37.test` is refused because the account already exists (Keycloak says: "efox = true; nava We are sorry... User with email lead-bafd0f2c@onb-d8bdba37.test already exists. Please login to account management to link the accou")
- **PASS** control: `admin@onb-d8bdba37.test` signs in through the same IdP of tenant C (no error text on the page)
- **PASS** after a group sync cycle `lead-bafd0f2c@onb-d8bdba37.test` is not in `/t/d8bdba37-661f-5ee8-9dab-3dddfe18a269/` and not a member of `t-d8bdba37-661f-5ee8-9dab-3dddfe18a269`
- **PASS** `lead-bafd0f2c@onb-d8bdba37.test` holds no link to tenant C's IdP
`lead-bafd0f2c@onb-d8bdba37.test` (signed in through B's IdP) names tenant C:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <lead-bafd0f2c@onb-d8bdba37.test token>
X-Tenant-Id: d8bdba37-661f-5ee8-9dab-3dddfe18a269
```

```http
HTTP/1.1 401

{"type":"urn:nava:problem:tenant_unresolved","title":"Tenant unresolved","status":401,"code":"tenant_unresolved","detail":"The request could not be tied to a tenant.","instance":"urn:nava:request:e93cd25f968e9e730ce8f1d49aaaae6f"}
```

- **PASS** `/me` never resolves tenant C (got 401, tenant `00000000-0000-0000-0000-000000000000`)
