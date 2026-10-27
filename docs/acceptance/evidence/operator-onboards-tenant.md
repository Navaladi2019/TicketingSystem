# S4 acceptance evidence (TICK-46)

Recorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on 2026-10-27T16:27:33Z. Tokens are never recorded; the IdP client secret is redacted.

Tenant A: `280f5bed-d3d0-5437-b735-ce1a19c82255`, domain `onb-280f5bed.test`, admin `admin@onb-280f5bed.test`.
Tenant B: `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, domain `onb-b04f75a5.test`, admin `admin@onb-b04f75a5.test`.


## Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing

First call, tenant onb-280f5bed.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 1d22faea-85a2-44d8-84ba-540854507785
Content-Type: application/json

{
  "admin_email": "admin@onb-280f5bed.test",
  "email_domains": [
    "onb-280f5bed.test"
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
    "display_name": "Onboarded onb-280f5bed.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"280f5bed-d3d0-5437-b735-ce1a19c82255"}
```

- **PASS** first call: 201, tenant_id `280f5bed-d3d0-5437-b735-ce1a19c82255`, state `active` (want 201, `280f5bed-d3d0-5437-b735-ce1a19c82255`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 1d22faea-85a2-44d8-84ba-540854507785
Content-Type: application/json

{
  "admin_email": "admin@onb-280f5bed.test",
  "email_domains": [
    "onb-280f5bed.test"
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
    "display_name": "Onboarded onb-280f5bed.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"280f5bed-d3d0-5437-b735-ce1a19c82255"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-280f5bed-d3d0-5437-b735-ce1a19c82255`, after `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-280f5bed-d3d0-5437-b735-ce1a19c82255`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 1d22faea-85a2-44d8-84ba-540854507785
Content-Type: application/json

{
  "admin_email": "admin@onb-280f5bed.test",
  "email_domains": [
    "onb-280f5bed.test"
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
    "display_name": "Changed onb-280f5bed.test",
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
- **PASS** the admin `admin@onb-280f5bed.test` has a pending invite and no Keycloak user yet
First call, tenant onb-b04f75a5.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 57bda71f-4730-43ef-bb24-cdae03a1df0b
Content-Type: application/json

{
  "admin_email": "admin@onb-b04f75a5.test",
  "email_domains": [
    "onb-b04f75a5.test"
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
    "display_name": "Onboarded onb-b04f75a5.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"b04f75a5-e0f7-53a6-aa30-723ebc343bf1"}
```

- **PASS** first call: 201, tenant_id `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, state `active` (want 201, `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 57bda71f-4730-43ef-bb24-cdae03a1df0b
Content-Type: application/json

{
  "admin_email": "admin@onb-b04f75a5.test",
  "email_domains": [
    "onb-b04f75a5.test"
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
    "display_name": "Onboarded onb-b04f75a5.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"b04f75a5-e0f7-53a6-aa30-723ebc343bf1"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, after `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 57bda71f-4730-43ef-bb24-cdae03a1df0b
Content-Type: application/json

{
  "admin_email": "admin@onb-b04f75a5.test",
  "email_domains": [
    "onb-b04f75a5.test"
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
    "display_name": "Changed onb-b04f75a5.test",
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
- **PASS** the admin `admin@onb-b04f75a5.test` has a pending invite and no Keycloak user yet

## Criterion 2: the new admin signs in and is scoped to the new tenant

- **PASS** `admin@onb-280f5bed.test` signed in through IdP `idp-280f5bed-d3d0-5437-b735-ce1a19c82255`; the token's `organization` claim is exactly `t-280f5bed-d3d0-5437-b735-ce1a19c82255`
- **PASS** `admin@onb-b04f75a5.test` signed in through IdP `idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`; the token's `organization` claim is exactly `t-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`
One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).

Admin of onb-280f5bed.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-280f5bed.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-280f5bed-d3d0-5437-b735-ce1a19c82255","tenant_id":"280f5bed-d3d0-5437-b735-ce1a19c82255","user_id":"b8fa730e-64ab-45d1-8a03-49ec19353c10"}
```

- **PASS** `/me` is tenant `280f5bed-d3d0-5437-b735-ce1a19c82255` via `idp-280f5bed-d3d0-5437-b735-ce1a19c82255` (got 200, tenant `280f5bed-d3d0-5437-b735-ce1a19c82255`, idp `idp-280f5bed-d3d0-5437-b735-ce1a19c82255`)
- **PASS** `admin@onb-280f5bed.test` is in `/t/280f5bed-d3d0-5437-b735-ce1a19c82255/` and the invite is consumed
Admin of onb-b04f75a5.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-b04f75a5.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1","tenant_id":"b04f75a5-e0f7-53a6-aa30-723ebc343bf1","user_id":"0a0019dc-86b1-4d17-beec-1c5ec22cfb51"}
```

- **PASS** `/me` is tenant `b04f75a5-e0f7-53a6-aa30-723ebc343bf1` via `idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1` (got 200, tenant `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, idp `idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`)
- **PASS** `admin@onb-b04f75a5.test` is in `/t/b04f75a5-e0f7-53a6-aa30-723ebc343bf1/` and the invite is consumed

## Isolation check: tenant B cannot see or change what onboarding created for tenant A

Admin of onb-b04f75a5.test names tenant onb-280f5bed.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-b04f75a5.test token>
X-Tenant-Id: 280f5bed-d3d0-5437-b735-ce1a19c82255
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1","tenant_id":"b04f75a5-e0f7-53a6-aa30-723ebc343bf1","user_id":"0a0019dc-86b1-4d17-beec-1c5ec22cfb51"}
```

- **PASS** still tenant `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, never `280f5bed-d3d0-5437-b735-ce1a19c82255`
Admin of onb-b04f75a5.test names tenant onb-280f5bed.test:

```http
GET /api/v1/me?tenant_id=280f5bed-d3d0-5437-b735-ce1a19c82255 (nava-core)
Authorization: Bearer <admin@onb-b04f75a5.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1","tenant_id":"b04f75a5-e0f7-53a6-aa30-723ebc343bf1","user_id":"0a0019dc-86b1-4d17-beec-1c5ec22cfb51"}
```

- **PASS** still tenant `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`, never `280f5bed-d3d0-5437-b735-ce1a19c82255`
- **PASS** `admin@onb-b04f75a5.test` is not in `/t/280f5bed-d3d0-5437-b735-ce1a19c82255/` and not a member of organization `t-280f5bed-d3d0-5437-b735-ce1a19c82255`
- **PASS** `admin@onb-b04f75a5.test` holds no link to `idp-280f5bed-d3d0-5437-b735-ce1a19c82255`
Admin of onb-280f5bed.test names tenant onb-b04f75a5.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-280f5bed.test token>
X-Tenant-Id: b04f75a5-e0f7-53a6-aa30-723ebc343bf1
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-280f5bed-d3d0-5437-b735-ce1a19c82255","tenant_id":"280f5bed-d3d0-5437-b735-ce1a19c82255","user_id":"b8fa730e-64ab-45d1-8a03-49ec19353c10"}
```

- **PASS** still tenant `280f5bed-d3d0-5437-b735-ce1a19c82255`, never `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`
Admin of onb-280f5bed.test names tenant onb-b04f75a5.test:

```http
GET /api/v1/me?tenant_id=b04f75a5-e0f7-53a6-aa30-723ebc343bf1 (nava-core)
Authorization: Bearer <admin@onb-280f5bed.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-280f5bed-d3d0-5437-b735-ce1a19c82255","tenant_id":"280f5bed-d3d0-5437-b735-ce1a19c82255","user_id":"b8fa730e-64ab-45d1-8a03-49ec19353c10"}
```

- **PASS** still tenant `280f5bed-d3d0-5437-b735-ce1a19c82255`, never `b04f75a5-e0f7-53a6-aa30-723ebc343bf1`
- **PASS** `admin@onb-280f5bed.test` is not in `/t/b04f75a5-e0f7-53a6-aa30-723ebc343bf1/` and not a member of organization `t-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`
- **PASS** `admin@onb-280f5bed.test` holds no link to `idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1`
A new tenant claims tenant A's domain:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 2b92810b-c66e-47c9-bfe4-0acf2eff96a9
Content-Type: application/json

{
  "admin_email": "admin@onb-280f5bed.test",
  "email_domains": [
    "onb-280f5bed.test"
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
    "display_name": "Onboarded onb-280f5bed.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 500

(no body: busybox wget keeps none for an error status)
```

- **FAIL** 500 (want 409 `domain_claimed`); `onb-280f5bed.test` is still owned by `280f5bed-d3d0-5437-b735-ce1a19c82255` (got `280f5bed-d3d0-5437-b735-ce1a19c82255`)
- **PASS** no second invite for `admin@onb-280f5bed.test`
Tenant B's admin replays A's onboarding on nava-core:

```http
POST /api/v1/operator/tenants (nava-core)
Authorization: Bearer <admin@onb-b04f75a5.test token>
Idempotency-Key: 1d22faea-85a2-44d8-84ba-540854507785
```

```http
HTTP/1.1 501

{"type":"urn:nava:problem:not_implemented","title":"Not implemented","status":501,"code":"not_implemented","detail":"This operation is declared in the contract but not implemented yet.","instance":"urn:nava:request:3269efcd39fb74f7d9302d003eb531f2"}
```

- **PASS** refused with 501; nothing changes for tenant A (`3 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-280f5bed-d3d0-5437-b735-ce1a19c82255`)

## Criterion 3: inviting a user already federated to another tenant's IdP is rejected

- **PASS** precondition: `lead-aa3f1cd1@onb-0421456e.test` exists and is federated to tenant B's IdP only (`[idp-b04f75a5-e0f7-53a6-aa30-723ebc343bf1]`)
Onboard tenant C naming that user as admin:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 19ae42ac-605d-4896-91d7-e2a2416aa8bc
Content-Type: application/json

{
  "admin_email": "lead-aa3f1cd1@onb-0421456e.test",
  "email_domains": [
    "onb-0421456e.test"
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
    "display_name": "Onboarded onb-0421456e.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"0421456e-8b7a-5b3e-918a-8f03d74d0d4d"}
```

Status 201. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.

- **PASS** no invite for `lead-aa3f1cd1@onb-0421456e.test` in tenant C (got `missing`)
- **PASS** signing in through tenant C's IdP as `lead-aa3f1cd1@onb-0421456e.test` is refused because the account already exists (Keycloak says: "efox = true; nava We are sorry... User with email lead-aa3f1cd1@onb-0421456e.test already exists. Please login to account management to link the accou")
- **PASS** control: `admin@onb-0421456e.test` signs in through the same IdP of tenant C (no error text on the page)
- **PASS** after a group sync cycle `lead-aa3f1cd1@onb-0421456e.test` is not in `/t/0421456e-8b7a-5b3e-918a-8f03d74d0d4d/` and not a member of `t-0421456e-8b7a-5b3e-918a-8f03d74d0d4d`
- **PASS** `lead-aa3f1cd1@onb-0421456e.test` holds no link to tenant C's IdP
`lead-aa3f1cd1@onb-0421456e.test` (signed in through B's IdP) names tenant C:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <lead-aa3f1cd1@onb-0421456e.test token>
X-Tenant-Id: 0421456e-8b7a-5b3e-918a-8f03d74d0d4d
```

```http
HTTP/1.1 401

{"type":"urn:nava:problem:tenant_unresolved","title":"Tenant unresolved","status":401,"code":"tenant_unresolved","detail":"The request could not be tied to a tenant.","instance":"urn:nava:request:cac71786d0b9c26afa62e6d04ea2d80b"}
```

- **PASS** `/me` never resolves tenant C (got 401, tenant `00000000-0000-0000-0000-000000000000`)
