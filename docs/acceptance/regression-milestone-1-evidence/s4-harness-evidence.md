# S4 acceptance evidence (TICK-46)

Recorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on 2026-10-28T17:17:11Z. Tokens are never recorded; the IdP client secret is redacted.

Tenant A: `2c47d9fc-ab82-5525-a7d9-86cac6273407`, domain `onb-2c47d9fc.test`, admin `admin@onb-2c47d9fc.test`.
Tenant B: `19985f68-d352-5894-b494-9ab176e8ae27`, domain `onb-19985f68.test`, admin `admin@onb-19985f68.test`.


## Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing

First call, tenant onb-2c47d9fc.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 013094f9-2d4f-4f2a-865c-6ebd3f2fbdc5
Content-Type: application/json

{
  "admin_email": "admin@onb-2c47d9fc.test",
  "email_domains": [
    "onb-2c47d9fc.test"
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
    "display_name": "Onboarded onb-2c47d9fc.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"2c47d9fc-ab82-5525-a7d9-86cac6273407"}
```

- **PASS** first call: 201, tenant_id `2c47d9fc-ab82-5525-a7d9-86cac6273407`, state `active` (want 201, `2c47d9fc-ab82-5525-a7d9-86cac6273407`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 013094f9-2d4f-4f2a-865c-6ebd3f2fbdc5
Content-Type: application/json

{
  "admin_email": "admin@onb-2c47d9fc.test",
  "email_domains": [
    "onb-2c47d9fc.test"
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
    "display_name": "Onboarded onb-2c47d9fc.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"2c47d9fc-ab82-5525-a7d9-86cac6273407"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `3 tenants, 5 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-2c47d9fc-ab82-5525-a7d9-86cac6273407`, after `3 tenants, 5 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-2c47d9fc-ab82-5525-a7d9-86cac6273407`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 013094f9-2d4f-4f2a-865c-6ebd3f2fbdc5
Content-Type: application/json

{
  "admin_email": "admin@onb-2c47d9fc.test",
  "email_domains": [
    "onb-2c47d9fc.test"
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
    "display_name": "Changed onb-2c47d9fc.test",
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
- **PASS** the admin `admin@onb-2c47d9fc.test` has a pending invite and no Keycloak user yet
First call, tenant onb-19985f68.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e49770e8-5e12-43d2-8660-3959d95e29b4
Content-Type: application/json

{
  "admin_email": "admin@onb-19985f68.test",
  "email_domains": [
    "onb-19985f68.test"
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
    "display_name": "Onboarded onb-19985f68.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"19985f68-d352-5894-b494-9ab176e8ae27"}
```

- **PASS** first call: 201, tenant_id `19985f68-d352-5894-b494-9ab176e8ae27`, state `active` (want 201, `19985f68-d352-5894-b494-9ab176e8ae27`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e49770e8-5e12-43d2-8660-3959d95e29b4
Content-Type: application/json

{
  "admin_email": "admin@onb-19985f68.test",
  "email_domains": [
    "onb-19985f68.test"
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
    "display_name": "Onboarded onb-19985f68.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"19985f68-d352-5894-b494-9ab176e8ae27"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `4 tenants, 6 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-19985f68-d352-5894-b494-9ab176e8ae27`, after `4 tenants, 6 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-19985f68-d352-5894-b494-9ab176e8ae27`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e49770e8-5e12-43d2-8660-3959d95e29b4
Content-Type: application/json

{
  "admin_email": "admin@onb-19985f68.test",
  "email_domains": [
    "onb-19985f68.test"
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
    "display_name": "Changed onb-19985f68.test",
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
- **PASS** the admin `admin@onb-19985f68.test` has a pending invite and no Keycloak user yet

## Criterion 2: the new admin signs in and is scoped to the new tenant

- **PASS** `admin@onb-2c47d9fc.test` signed in through IdP `idp-2c47d9fc-ab82-5525-a7d9-86cac6273407`; the token's `organization` claim is exactly `t-2c47d9fc-ab82-5525-a7d9-86cac6273407`
- **PASS** `admin@onb-19985f68.test` signed in through IdP `idp-19985f68-d352-5894-b494-9ab176e8ae27`; the token's `organization` claim is exactly `t-19985f68-d352-5894-b494-9ab176e8ae27`
One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).

Admin of onb-2c47d9fc.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-2c47d9fc.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-2c47d9fc-ab82-5525-a7d9-86cac6273407","tenant_id":"2c47d9fc-ab82-5525-a7d9-86cac6273407","user_id":"4613935d-001b-4a3c-9029-205ab0bbad25"}
```

- **PASS** `/me` is tenant `2c47d9fc-ab82-5525-a7d9-86cac6273407` via `idp-2c47d9fc-ab82-5525-a7d9-86cac6273407` (got 200, tenant `2c47d9fc-ab82-5525-a7d9-86cac6273407`, idp `idp-2c47d9fc-ab82-5525-a7d9-86cac6273407`)
- **PASS** `admin@onb-2c47d9fc.test` is in `/t/2c47d9fc-ab82-5525-a7d9-86cac6273407/` and the invite is consumed
Admin of onb-19985f68.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-19985f68.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-19985f68-d352-5894-b494-9ab176e8ae27","tenant_id":"19985f68-d352-5894-b494-9ab176e8ae27","user_id":"9acc3ab1-2f39-4d73-9f76-18aa3c794831"}
```

- **PASS** `/me` is tenant `19985f68-d352-5894-b494-9ab176e8ae27` via `idp-19985f68-d352-5894-b494-9ab176e8ae27` (got 200, tenant `19985f68-d352-5894-b494-9ab176e8ae27`, idp `idp-19985f68-d352-5894-b494-9ab176e8ae27`)
- **PASS** `admin@onb-19985f68.test` is in `/t/19985f68-d352-5894-b494-9ab176e8ae27/` and the invite is consumed

## Isolation check: tenant B cannot see or change what onboarding created for tenant A

Admin of onb-19985f68.test names tenant onb-2c47d9fc.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-19985f68.test token>
X-Tenant-Id: 2c47d9fc-ab82-5525-a7d9-86cac6273407
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-19985f68-d352-5894-b494-9ab176e8ae27","tenant_id":"19985f68-d352-5894-b494-9ab176e8ae27","user_id":"9acc3ab1-2f39-4d73-9f76-18aa3c794831"}
```

- **PASS** still tenant `19985f68-d352-5894-b494-9ab176e8ae27`, never `2c47d9fc-ab82-5525-a7d9-86cac6273407`
Admin of onb-19985f68.test names tenant onb-2c47d9fc.test:

```http
GET /api/v1/me?tenant_id=2c47d9fc-ab82-5525-a7d9-86cac6273407 (nava-core)
Authorization: Bearer <admin@onb-19985f68.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-19985f68-d352-5894-b494-9ab176e8ae27","tenant_id":"19985f68-d352-5894-b494-9ab176e8ae27","user_id":"9acc3ab1-2f39-4d73-9f76-18aa3c794831"}
```

- **PASS** still tenant `19985f68-d352-5894-b494-9ab176e8ae27`, never `2c47d9fc-ab82-5525-a7d9-86cac6273407`
- **PASS** `admin@onb-19985f68.test` is not in `/t/2c47d9fc-ab82-5525-a7d9-86cac6273407/` and not a member of organization `t-2c47d9fc-ab82-5525-a7d9-86cac6273407`
- **PASS** `admin@onb-19985f68.test` holds no link to `idp-2c47d9fc-ab82-5525-a7d9-86cac6273407`
Admin of onb-2c47d9fc.test names tenant onb-19985f68.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-2c47d9fc.test token>
X-Tenant-Id: 19985f68-d352-5894-b494-9ab176e8ae27
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-2c47d9fc-ab82-5525-a7d9-86cac6273407","tenant_id":"2c47d9fc-ab82-5525-a7d9-86cac6273407","user_id":"4613935d-001b-4a3c-9029-205ab0bbad25"}
```

- **PASS** still tenant `2c47d9fc-ab82-5525-a7d9-86cac6273407`, never `19985f68-d352-5894-b494-9ab176e8ae27`
Admin of onb-2c47d9fc.test names tenant onb-19985f68.test:

```http
GET /api/v1/me?tenant_id=19985f68-d352-5894-b494-9ab176e8ae27 (nava-core)
Authorization: Bearer <admin@onb-2c47d9fc.test token>
```

```http
HTTP/1.1 200

{"group_paths":[],"groups":[],"idp_alias":"idp-2c47d9fc-ab82-5525-a7d9-86cac6273407","tenant_id":"2c47d9fc-ab82-5525-a7d9-86cac6273407","user_id":"4613935d-001b-4a3c-9029-205ab0bbad25"}
```

- **PASS** still tenant `2c47d9fc-ab82-5525-a7d9-86cac6273407`, never `19985f68-d352-5894-b494-9ab176e8ae27`
- **PASS** `admin@onb-2c47d9fc.test` is not in `/t/19985f68-d352-5894-b494-9ab176e8ae27/` and not a member of organization `t-19985f68-d352-5894-b494-9ab176e8ae27`
- **PASS** `admin@onb-2c47d9fc.test` holds no link to `idp-19985f68-d352-5894-b494-9ab176e8ae27`
A new tenant claims tenant A's domain:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 2fa61bc6-7fee-4a35-af69-58c06507e6bc
Content-Type: application/json

{
  "admin_email": "admin@onb-2c47d9fc.test",
  "email_domains": [
    "onb-2c47d9fc.test"
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
    "display_name": "Onboarded onb-2c47d9fc.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

{"type":"urn:nava:problem:domain_claimed","title":"Email domain already claimed","status":409,"code":"domain_claimed","detail":"An email domain of the request is already claimed by a tenant.","instance":"urn:nava:request:eec4b072868e3a7b2fbddd02dafe169b"}
```

- **PASS** 409 `domain_claimed` (want 409 `domain_claimed`)
- **PASS** the body names neither tenant A (`2c47d9fc-ab82-5525-a7d9-86cac6273407`) nor its organization (`t-2c47d9fc-ab82-5525-a7d9-86cac6273407`, id `755cb588-8b5c-4640-8adf-1d5505edab8e`) nor its IdP
- **PASS** `onb-2c47d9fc.test` is still owned by tenant A (got `2c47d9fc-ab82-5525-a7d9-86cac6273407`)
- **PASS** no `tenants` row (provisioning or otherwise) for the refused tenant `596e0f48-6868-5f39-a069-e1e23fdbfd60` (got 0)
- **PASS** no Keycloak organization `t-596e0f48-6868-5f39-a069-e1e23fdbfd60` (got [])
- **PASS** no `tenant_email_domain` row for the refused tenant (got 0)
- **PASS** no second invite for `admin@onb-2c47d9fc.test`
Tenant B's admin replays A's onboarding on nava-core:

```http
POST /api/v1/operator/tenants (nava-core)
Authorization: Bearer <admin@onb-19985f68.test token>
Idempotency-Key: 013094f9-2d4f-4f2a-865c-6ebd3f2fbdc5
```

```http
HTTP/1.1 501

{"type":"urn:nava:problem:not_implemented","title":"Not implemented","status":501,"code":"not_implemented","detail":"This operation is declared in the contract but not implemented yet.","instance":"urn:nava:request:4af7e74a0bd892e8e89c7bb5a805d604"}
```

- **PASS** refused with 501; nothing changes for tenant A (`4 tenants, 6 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-2c47d9fc-ab82-5525-a7d9-86cac6273407`)

## Criterion 3: inviting a user already federated to another tenant's IdP is rejected

- **PASS** precondition: `lead-6d18f8f0@onb-b7a11e16.test` exists and is federated to tenant B's IdP only (`[idp-19985f68-d352-5894-b494-9ab176e8ae27]`)
Onboard tenant C naming that user as admin:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 78d538ab-98dd-431f-ac5e-ce786674b7a0
Content-Type: application/json

{
  "admin_email": "lead-6d18f8f0@onb-b7a11e16.test",
  "email_domains": [
    "onb-b7a11e16.test"
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
    "display_name": "Onboarded onb-b7a11e16.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"b7a11e16-a50b-5f5a-b9bd-637f5ee2811e"}
```

Status 201. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.

- **PASS** no invite for `lead-6d18f8f0@onb-b7a11e16.test` in tenant C (got `missing`)
- **PASS** signing in through tenant C's IdP as `lead-6d18f8f0@onb-b7a11e16.test` is refused because the account already exists (Keycloak says: "efox = true; nava We are sorry... User with email lead-6d18f8f0@onb-b7a11e16.test already exists. Please login to account management to link the accou")
- **PASS** control: `admin@onb-b7a11e16.test` signs in through the same IdP of tenant C (no error text on the page)
- **PASS** after a group sync cycle `lead-6d18f8f0@onb-b7a11e16.test` is not in `/t/b7a11e16-a50b-5f5a-b9bd-637f5ee2811e/` and not a member of `t-b7a11e16-a50b-5f5a-b9bd-637f5ee2811e`
- **PASS** `lead-6d18f8f0@onb-b7a11e16.test` holds no link to tenant C's IdP
`lead-6d18f8f0@onb-b7a11e16.test` (signed in through B's IdP) names tenant C:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <lead-6d18f8f0@onb-b7a11e16.test token>
X-Tenant-Id: b7a11e16-a50b-5f5a-b9bd-637f5ee2811e
```

```http
HTTP/1.1 401

{"type":"urn:nava:problem:tenant_unresolved","title":"Tenant unresolved","status":401,"code":"tenant_unresolved","detail":"The request could not be tied to a tenant.","instance":"urn:nava:request:f5b60ce5111523113f2ff374d46dfd2d"}
```

- **PASS** `/me` never resolves tenant C (got 401, tenant `00000000-0000-0000-0000-000000000000`)
