# S4 acceptance evidence (TICK-46)

Recorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on 2026-10-27T23:29:22Z. Tokens are never recorded; the IdP client secret is redacted.

Tenant A: `04406bef-2a3e-5698-9ce8-6e79c88968a2`, domain `onb-04406bef.test`, admin `admin@onb-04406bef.test`.
Tenant B: `3c10c663-6833-5bea-96ff-1eb9f1201e72`, domain `onb-3c10c663.test`, admin `admin@onb-3c10c663.test`.


## Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing

First call, tenant onb-04406bef.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 8dcc6eda-1d8e-45b9-b0ca-856a05ab8264
Content-Type: application/json

{
  "admin_email": "admin@onb-04406bef.test",
  "email_domains": [
    "onb-04406bef.test"
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
    "display_name": "Onboarded onb-04406bef.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"04406bef-2a3e-5698-9ce8-6e79c88968a2"}
```

- **PASS** first call: 201, tenant_id `04406bef-2a3e-5698-9ce8-6e79c88968a2`, state `active` (want 201, `04406bef-2a3e-5698-9ce8-6e79c88968a2`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 8dcc6eda-1d8e-45b9-b0ca-856a05ab8264
Content-Type: application/json

{
  "admin_email": "admin@onb-04406bef.test",
  "email_domains": [
    "onb-04406bef.test"
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
    "display_name": "Onboarded onb-04406bef.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"04406bef-2a3e-5698-9ce8-6e79c88968a2"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-04406bef-2a3e-5698-9ce8-6e79c88968a2`, after `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-04406bef-2a3e-5698-9ce8-6e79c88968a2`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 8dcc6eda-1d8e-45b9-b0ca-856a05ab8264
Content-Type: application/json

{
  "admin_email": "admin@onb-04406bef.test",
  "email_domains": [
    "onb-04406bef.test"
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
    "display_name": "Changed onb-04406bef.test",
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
- **PASS** the admin `admin@onb-04406bef.test` has a pending invite and no Keycloak user yet
First call, tenant onb-3c10c663.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: b1ede569-5148-4a47-a9bc-17c68b48b9c3
Content-Type: application/json

{
  "admin_email": "admin@onb-3c10c663.test",
  "email_domains": [
    "onb-3c10c663.test"
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
    "display_name": "Onboarded onb-3c10c663.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"3c10c663-6833-5bea-96ff-1eb9f1201e72"}
```

- **PASS** first call: 201, tenant_id `3c10c663-6833-5bea-96ff-1eb9f1201e72`, state `active` (want 201, `3c10c663-6833-5bea-96ff-1eb9f1201e72`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: b1ede569-5148-4a47-a9bc-17c68b48b9c3
Content-Type: application/json

{
  "admin_email": "admin@onb-3c10c663.test",
  "email_domains": [
    "onb-3c10c663.test"
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
    "display_name": "Onboarded onb-3c10c663.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"3c10c663-6833-5bea-96ff-1eb9f1201e72"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-3c10c663-6833-5bea-96ff-1eb9f1201e72`, after `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-3c10c663-6833-5bea-96ff-1eb9f1201e72`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: b1ede569-5148-4a47-a9bc-17c68b48b9c3
Content-Type: application/json

{
  "admin_email": "admin@onb-3c10c663.test",
  "email_domains": [
    "onb-3c10c663.test"
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
    "display_name": "Changed onb-3c10c663.test",
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
- **PASS** the admin `admin@onb-3c10c663.test` has a pending invite and no Keycloak user yet

## Criterion 2: the new admin signs in and is scoped to the new tenant

- **PASS** `admin@onb-04406bef.test` signed in through IdP `idp-04406bef-2a3e-5698-9ce8-6e79c88968a2`; the token's `organization` claim is exactly `t-04406bef-2a3e-5698-9ce8-6e79c88968a2`
- **PASS** `admin@onb-3c10c663.test` signed in through IdP `idp-3c10c663-6833-5bea-96ff-1eb9f1201e72`; the token's `organization` claim is exactly `t-3c10c663-6833-5bea-96ff-1eb9f1201e72`
One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).

Admin of onb-04406bef.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-04406bef.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-04406bef-2a3e-5698-9ce8-6e79c88968a2","tenant_id":"04406bef-2a3e-5698-9ce8-6e79c88968a2","user_id":"6fe48e19-a350-47ff-9859-aa2b26a50b12"}
```

- **PASS** `/me` is tenant `04406bef-2a3e-5698-9ce8-6e79c88968a2` via `idp-04406bef-2a3e-5698-9ce8-6e79c88968a2` (got 200, tenant `04406bef-2a3e-5698-9ce8-6e79c88968a2`, idp `idp-04406bef-2a3e-5698-9ce8-6e79c88968a2`)
- **PASS** `admin@onb-04406bef.test` is in `/t/04406bef-2a3e-5698-9ce8-6e79c88968a2/` and the invite is consumed
Admin of onb-3c10c663.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-3c10c663.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-3c10c663-6833-5bea-96ff-1eb9f1201e72","tenant_id":"3c10c663-6833-5bea-96ff-1eb9f1201e72","user_id":"287d9116-9b3c-4698-959b-c4612180be1a"}
```

- **PASS** `/me` is tenant `3c10c663-6833-5bea-96ff-1eb9f1201e72` via `idp-3c10c663-6833-5bea-96ff-1eb9f1201e72` (got 200, tenant `3c10c663-6833-5bea-96ff-1eb9f1201e72`, idp `idp-3c10c663-6833-5bea-96ff-1eb9f1201e72`)
- **PASS** `admin@onb-3c10c663.test` is in `/t/3c10c663-6833-5bea-96ff-1eb9f1201e72/` and the invite is consumed

## Isolation check: tenant B cannot see or change what onboarding created for tenant A

Admin of onb-3c10c663.test names tenant onb-04406bef.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-3c10c663.test token>
X-Tenant-Id: 04406bef-2a3e-5698-9ce8-6e79c88968a2
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-3c10c663-6833-5bea-96ff-1eb9f1201e72","tenant_id":"3c10c663-6833-5bea-96ff-1eb9f1201e72","user_id":"287d9116-9b3c-4698-959b-c4612180be1a"}
```

- **PASS** still tenant `3c10c663-6833-5bea-96ff-1eb9f1201e72`, never `04406bef-2a3e-5698-9ce8-6e79c88968a2`
Admin of onb-3c10c663.test names tenant onb-04406bef.test:

```http
GET /api/v1/me?tenant_id=04406bef-2a3e-5698-9ce8-6e79c88968a2 (nava-core)
Authorization: Bearer <admin@onb-3c10c663.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-3c10c663-6833-5bea-96ff-1eb9f1201e72","tenant_id":"3c10c663-6833-5bea-96ff-1eb9f1201e72","user_id":"287d9116-9b3c-4698-959b-c4612180be1a"}
```

- **PASS** still tenant `3c10c663-6833-5bea-96ff-1eb9f1201e72`, never `04406bef-2a3e-5698-9ce8-6e79c88968a2`
- **PASS** `admin@onb-3c10c663.test` is not in `/t/04406bef-2a3e-5698-9ce8-6e79c88968a2/` and not a member of organization `t-04406bef-2a3e-5698-9ce8-6e79c88968a2`
- **PASS** `admin@onb-3c10c663.test` holds no link to `idp-04406bef-2a3e-5698-9ce8-6e79c88968a2`
Admin of onb-04406bef.test names tenant onb-3c10c663.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-04406bef.test token>
X-Tenant-Id: 3c10c663-6833-5bea-96ff-1eb9f1201e72
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-04406bef-2a3e-5698-9ce8-6e79c88968a2","tenant_id":"04406bef-2a3e-5698-9ce8-6e79c88968a2","user_id":"6fe48e19-a350-47ff-9859-aa2b26a50b12"}
```

- **PASS** still tenant `04406bef-2a3e-5698-9ce8-6e79c88968a2`, never `3c10c663-6833-5bea-96ff-1eb9f1201e72`
Admin of onb-04406bef.test names tenant onb-3c10c663.test:

```http
GET /api/v1/me?tenant_id=3c10c663-6833-5bea-96ff-1eb9f1201e72 (nava-core)
Authorization: Bearer <admin@onb-04406bef.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-04406bef-2a3e-5698-9ce8-6e79c88968a2","tenant_id":"04406bef-2a3e-5698-9ce8-6e79c88968a2","user_id":"6fe48e19-a350-47ff-9859-aa2b26a50b12"}
```

- **PASS** still tenant `04406bef-2a3e-5698-9ce8-6e79c88968a2`, never `3c10c663-6833-5bea-96ff-1eb9f1201e72`
- **PASS** `admin@onb-04406bef.test` is not in `/t/3c10c663-6833-5bea-96ff-1eb9f1201e72/` and not a member of organization `t-3c10c663-6833-5bea-96ff-1eb9f1201e72`
- **PASS** `admin@onb-04406bef.test` holds no link to `idp-3c10c663-6833-5bea-96ff-1eb9f1201e72`
A new tenant claims tenant A's domain:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: a0eba041-b6d0-4191-ba7f-877646a17fbc
Content-Type: application/json

{
  "admin_email": "admin@onb-04406bef.test",
  "email_domains": [
    "onb-04406bef.test"
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
    "display_name": "Onboarded onb-04406bef.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

{"type":"urn:nava:problem:domain_claimed","title":"Email domain already claimed","status":409,"code":"domain_claimed","detail":"An email domain of the request is already claimed by a tenant.","instance":"urn:nava:request:bd135e5c03ab013bf516e3549cadcc3c"}
```

- **PASS** 409 `domain_claimed` (want 409 `domain_claimed`)
- **PASS** the body names neither tenant A (`04406bef-2a3e-5698-9ce8-6e79c88968a2`) nor its organization (`t-04406bef-2a3e-5698-9ce8-6e79c88968a2`, id `7da4873b-8001-40cc-9d1f-d2051492f966`) nor its IdP
- **PASS** `onb-04406bef.test` is still owned by tenant A (got `04406bef-2a3e-5698-9ce8-6e79c88968a2`)
- **PASS** no `tenants` row (provisioning or otherwise) for the refused tenant `5597daa7-c51a-5fa1-9433-2822a19960f0` (got 0)
- **PASS** no Keycloak organization `t-5597daa7-c51a-5fa1-9433-2822a19960f0` (got [])
- **PASS** no `tenant_email_domain` row for the refused tenant (got 0)
- **PASS** no second invite for `admin@onb-04406bef.test`
Tenant B's admin replays A's onboarding on nava-core:

```http
POST /api/v1/operator/tenants (nava-core)
Authorization: Bearer <admin@onb-3c10c663.test token>
Idempotency-Key: 8dcc6eda-1d8e-45b9-b0ca-856a05ab8264
```

```http
HTTP/1.1 501

{"type":"urn:nava:problem:not_implemented","title":"Not implemented","status":501,"code":"not_implemented","detail":"This operation is declared in the contract but not implemented yet.","instance":"urn:nava:request:6c00fd266872381c44a3abe2b2ea93a9"}
```

- **PASS** refused with 501; nothing changes for tenant A (`2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-04406bef-2a3e-5698-9ce8-6e79c88968a2`)

## Criterion 3: inviting a user already federated to another tenant's IdP is rejected

- **PASS** precondition: `lead-3ccf9c8a@onb-09676af6.test` exists and is federated to tenant B's IdP only (`[idp-3c10c663-6833-5bea-96ff-1eb9f1201e72]`)
Onboard tenant C naming that user as admin:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 3e063cc3-b857-4033-86eb-71b92a645b10
Content-Type: application/json

{
  "admin_email": "lead-3ccf9c8a@onb-09676af6.test",
  "email_domains": [
    "onb-09676af6.test"
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
    "display_name": "Onboarded onb-09676af6.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"09676af6-99d9-54b8-ae2e-2fdc73b27a35"}
```

Status 201. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.

- **PASS** no invite for `lead-3ccf9c8a@onb-09676af6.test` in tenant C (got `missing`)
- **PASS** signing in through tenant C's IdP as `lead-3ccf9c8a@onb-09676af6.test` is refused because the account already exists (Keycloak says: "efox = true; nava We are sorry... User with email lead-3ccf9c8a@onb-09676af6.test already exists. Please login to account management to link the accou")
- **PASS** control: `admin@onb-09676af6.test` signs in through the same IdP of tenant C (no error text on the page)
- **PASS** after a group sync cycle `lead-3ccf9c8a@onb-09676af6.test` is not in `/t/09676af6-99d9-54b8-ae2e-2fdc73b27a35/` and not a member of `t-09676af6-99d9-54b8-ae2e-2fdc73b27a35`
- **PASS** `lead-3ccf9c8a@onb-09676af6.test` holds no link to tenant C's IdP
`lead-3ccf9c8a@onb-09676af6.test` (signed in through B's IdP) names tenant C:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <lead-3ccf9c8a@onb-09676af6.test token>
X-Tenant-Id: 09676af6-99d9-54b8-ae2e-2fdc73b27a35
```

```http
HTTP/1.1 401

{"type":"urn:nava:problem:tenant_unresolved","title":"Tenant unresolved","status":401,"code":"tenant_unresolved","detail":"The request could not be tied to a tenant.","instance":"urn:nava:request:6159f109d8ab4ee31e4f9d1b3fe6b6b1"}
```

- **PASS** `/me` never resolves tenant C (got 401, tenant `00000000-0000-0000-0000-000000000000`)
