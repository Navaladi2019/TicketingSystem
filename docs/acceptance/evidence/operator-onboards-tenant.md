# S4 acceptance evidence (TICK-46)

Recorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on 2026-10-27T21:21:20Z. Tokens are never recorded; the IdP client secret is redacted.

Tenant A: `bbb083da-f46c-5ea4-afe3-d844c007e16d`, domain `onb-bbb083da.test`, admin `admin@onb-bbb083da.test`.
Tenant B: `89f34f39-3b83-59b0-9000-5d62fe331118`, domain `onb-89f34f39.test`, admin `admin@onb-89f34f39.test`.


## Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing

First call, tenant onb-bbb083da.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e5b0895b-0ede-4a87-b0fd-94db8f00c2e2
Content-Type: application/json

{
  "admin_email": "admin@onb-bbb083da.test",
  "email_domains": [
    "onb-bbb083da.test"
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
    "display_name": "Onboarded onb-bbb083da.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"bbb083da-f46c-5ea4-afe3-d844c007e16d"}
```

- **PASS** first call: 201, tenant_id `bbb083da-f46c-5ea4-afe3-d844c007e16d`, state `active` (want 201, `bbb083da-f46c-5ea4-afe3-d844c007e16d`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e5b0895b-0ede-4a87-b0fd-94db8f00c2e2
Content-Type: application/json

{
  "admin_email": "admin@onb-bbb083da.test",
  "email_domains": [
    "onb-bbb083da.test"
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
    "display_name": "Onboarded onb-bbb083da.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"bbb083da-f46c-5ea4-afe3-d844c007e16d"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-bbb083da-f46c-5ea4-afe3-d844c007e16d`, after `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-bbb083da-f46c-5ea4-afe3-d844c007e16d`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: e5b0895b-0ede-4a87-b0fd-94db8f00c2e2
Content-Type: application/json

{
  "admin_email": "admin@onb-bbb083da.test",
  "email_domains": [
    "onb-bbb083da.test"
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
    "display_name": "Changed onb-bbb083da.test",
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
- **PASS** the admin `admin@onb-bbb083da.test` has a pending invite and no Keycloak user yet
First call, tenant onb-89f34f39.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: cf448165-f42e-4e2b-b500-4712c404feed
Content-Type: application/json

{
  "admin_email": "admin@onb-89f34f39.test",
  "email_domains": [
    "onb-89f34f39.test"
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
    "display_name": "Onboarded onb-89f34f39.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"89f34f39-3b83-59b0-9000-5d62fe331118"}
```

- **PASS** first call: 201, tenant_id `89f34f39-3b83-59b0-9000-5d62fe331118`, state `active` (want 201, `89f34f39-3b83-59b0-9000-5d62fe331118`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: cf448165-f42e-4e2b-b500-4712c404feed
Content-Type: application/json

{
  "admin_email": "admin@onb-89f34f39.test",
  "email_domains": [
    "onb-89f34f39.test"
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
    "display_name": "Onboarded onb-89f34f39.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"89f34f39-3b83-59b0-9000-5d62fe331118"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-89f34f39-3b83-59b0-9000-5d62fe331118`, after `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-89f34f39-3b83-59b0-9000-5d62fe331118`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: cf448165-f42e-4e2b-b500-4712c404feed
Content-Type: application/json

{
  "admin_email": "admin@onb-89f34f39.test",
  "email_domains": [
    "onb-89f34f39.test"
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
    "display_name": "Changed onb-89f34f39.test",
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
- **PASS** the admin `admin@onb-89f34f39.test` has a pending invite and no Keycloak user yet

## Criterion 2: the new admin signs in and is scoped to the new tenant

- **PASS** `admin@onb-bbb083da.test` signed in through IdP `idp-bbb083da-f46c-5ea4-afe3-d844c007e16d`; the token's `organization` claim is exactly `t-bbb083da-f46c-5ea4-afe3-d844c007e16d`
- **PASS** `admin@onb-89f34f39.test` signed in through IdP `idp-89f34f39-3b83-59b0-9000-5d62fe331118`; the token's `organization` claim is exactly `t-89f34f39-3b83-59b0-9000-5d62fe331118`
One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).

Admin of onb-bbb083da.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-bbb083da.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-bbb083da-f46c-5ea4-afe3-d844c007e16d","tenant_id":"bbb083da-f46c-5ea4-afe3-d844c007e16d","user_id":"762c9186-11a7-45d4-ab84-4808eb7f0e2d"}
```

- **PASS** `/me` is tenant `bbb083da-f46c-5ea4-afe3-d844c007e16d` via `idp-bbb083da-f46c-5ea4-afe3-d844c007e16d` (got 200, tenant `bbb083da-f46c-5ea4-afe3-d844c007e16d`, idp `idp-bbb083da-f46c-5ea4-afe3-d844c007e16d`)
- **PASS** `admin@onb-bbb083da.test` is in `/t/bbb083da-f46c-5ea4-afe3-d844c007e16d/` and the invite is consumed
Admin of onb-89f34f39.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-89f34f39.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-89f34f39-3b83-59b0-9000-5d62fe331118","tenant_id":"89f34f39-3b83-59b0-9000-5d62fe331118","user_id":"76201aad-325d-4ea9-8b45-e1b00675d756"}
```

- **PASS** `/me` is tenant `89f34f39-3b83-59b0-9000-5d62fe331118` via `idp-89f34f39-3b83-59b0-9000-5d62fe331118` (got 200, tenant `89f34f39-3b83-59b0-9000-5d62fe331118`, idp `idp-89f34f39-3b83-59b0-9000-5d62fe331118`)
- **PASS** `admin@onb-89f34f39.test` is in `/t/89f34f39-3b83-59b0-9000-5d62fe331118/` and the invite is consumed

## Isolation check: tenant B cannot see or change what onboarding created for tenant A

Admin of onb-89f34f39.test names tenant onb-bbb083da.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-89f34f39.test token>
X-Tenant-Id: bbb083da-f46c-5ea4-afe3-d844c007e16d
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-89f34f39-3b83-59b0-9000-5d62fe331118","tenant_id":"89f34f39-3b83-59b0-9000-5d62fe331118","user_id":"76201aad-325d-4ea9-8b45-e1b00675d756"}
```

- **PASS** still tenant `89f34f39-3b83-59b0-9000-5d62fe331118`, never `bbb083da-f46c-5ea4-afe3-d844c007e16d`
Admin of onb-89f34f39.test names tenant onb-bbb083da.test:

```http
GET /api/v1/me?tenant_id=bbb083da-f46c-5ea4-afe3-d844c007e16d (nava-core)
Authorization: Bearer <admin@onb-89f34f39.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-89f34f39-3b83-59b0-9000-5d62fe331118","tenant_id":"89f34f39-3b83-59b0-9000-5d62fe331118","user_id":"76201aad-325d-4ea9-8b45-e1b00675d756"}
```

- **PASS** still tenant `89f34f39-3b83-59b0-9000-5d62fe331118`, never `bbb083da-f46c-5ea4-afe3-d844c007e16d`
- **PASS** `admin@onb-89f34f39.test` is not in `/t/bbb083da-f46c-5ea4-afe3-d844c007e16d/` and not a member of organization `t-bbb083da-f46c-5ea4-afe3-d844c007e16d`
- **PASS** `admin@onb-89f34f39.test` holds no link to `idp-bbb083da-f46c-5ea4-afe3-d844c007e16d`
Admin of onb-bbb083da.test names tenant onb-89f34f39.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-bbb083da.test token>
X-Tenant-Id: 89f34f39-3b83-59b0-9000-5d62fe331118
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-bbb083da-f46c-5ea4-afe3-d844c007e16d","tenant_id":"bbb083da-f46c-5ea4-afe3-d844c007e16d","user_id":"762c9186-11a7-45d4-ab84-4808eb7f0e2d"}
```

- **PASS** still tenant `bbb083da-f46c-5ea4-afe3-d844c007e16d`, never `89f34f39-3b83-59b0-9000-5d62fe331118`
Admin of onb-bbb083da.test names tenant onb-89f34f39.test:

```http
GET /api/v1/me?tenant_id=89f34f39-3b83-59b0-9000-5d62fe331118 (nava-core)
Authorization: Bearer <admin@onb-bbb083da.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-bbb083da-f46c-5ea4-afe3-d844c007e16d","tenant_id":"bbb083da-f46c-5ea4-afe3-d844c007e16d","user_id":"762c9186-11a7-45d4-ab84-4808eb7f0e2d"}
```

- **PASS** still tenant `bbb083da-f46c-5ea4-afe3-d844c007e16d`, never `89f34f39-3b83-59b0-9000-5d62fe331118`
- **PASS** `admin@onb-bbb083da.test` is not in `/t/89f34f39-3b83-59b0-9000-5d62fe331118/` and not a member of organization `t-89f34f39-3b83-59b0-9000-5d62fe331118`
- **PASS** `admin@onb-bbb083da.test` holds no link to `idp-89f34f39-3b83-59b0-9000-5d62fe331118`
A new tenant claims tenant A's domain:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 01060071-9a58-4904-beee-9344c1110dde
Content-Type: application/json

{
  "admin_email": "admin@onb-bbb083da.test",
  "email_domains": [
    "onb-bbb083da.test"
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
    "display_name": "Onboarded onb-bbb083da.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

{"type":"urn:nava:problem:domain_claimed","title":"Email domain already claimed","status":409,"code":"domain_claimed","detail":"An email domain of the request is already claimed by a tenant.","instance":"urn:nava:request:67d9ea77ebe051e11f317b95b1f4ea7c"}
```

- **PASS** 409 `domain_claimed` (want 409 `domain_claimed`)
- **PASS** the body names neither tenant A (`bbb083da-f46c-5ea4-afe3-d844c007e16d`) nor its organization (`t-bbb083da-f46c-5ea4-afe3-d844c007e16d`, id `56f0ad23-448d-4506-96ff-9d5b96a28fdc`) nor its IdP
- **PASS** `onb-bbb083da.test` is still owned by tenant A (got `bbb083da-f46c-5ea4-afe3-d844c007e16d`)
- **PASS** no `tenants` row (provisioning or otherwise) for the refused tenant `2366ef6b-254b-5919-8f53-0fb2980f16a4` (got 0)
- **PASS** no Keycloak organization `t-2366ef6b-254b-5919-8f53-0fb2980f16a4` (got [])
- **PASS** no second invite for `admin@onb-bbb083da.test`
Tenant B's admin replays A's onboarding on nava-core:

```http
POST /api/v1/operator/tenants (nava-core)
Authorization: Bearer <admin@onb-89f34f39.test token>
Idempotency-Key: e5b0895b-0ede-4a87-b0fd-94db8f00c2e2
```

```http
HTTP/1.1 501

{"type":"urn:nava:problem:not_implemented","title":"Not implemented","status":501,"code":"not_implemented","detail":"This operation is declared in the contract but not implemented yet.","instance":"urn:nava:request:394bb8e7d033c47519fbd6f78b641220"}
```

- **PASS** refused with 501; nothing changes for tenant A (`2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-bbb083da-f46c-5ea4-afe3-d844c007e16d`)

## Criterion 3: inviting a user already federated to another tenant's IdP is rejected

- **PASS** precondition: `lead-bb1e4165@onb-aca5b34b.test` exists and is federated to tenant B's IdP only (`[idp-89f34f39-3b83-59b0-9000-5d62fe331118]`)
Onboard tenant C naming that user as admin:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 0931defe-1583-46ff-af33-542fe7fca8cc
Content-Type: application/json

{
  "admin_email": "lead-bb1e4165@onb-aca5b34b.test",
  "email_domains": [
    "onb-aca5b34b.test"
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
    "display_name": "Onboarded onb-aca5b34b.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"aca5b34b-183f-5a7c-bd8d-aec2f9261d51"}
```

Status 201. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.

- **PASS** no invite for `lead-bb1e4165@onb-aca5b34b.test` in tenant C (got `missing`)
- **PASS** signing in through tenant C's IdP as `lead-bb1e4165@onb-aca5b34b.test` is refused because the account already exists (Keycloak says: "efox = true; nava We are sorry... User with email lead-bb1e4165@onb-aca5b34b.test already exists. Please login to account management to link the accou")
- **PASS** control: `admin@onb-aca5b34b.test` signs in through the same IdP of tenant C (no error text on the page)
- **PASS** after a group sync cycle `lead-bb1e4165@onb-aca5b34b.test` is not in `/t/aca5b34b-183f-5a7c-bd8d-aec2f9261d51/` and not a member of `t-aca5b34b-183f-5a7c-bd8d-aec2f9261d51`
- **PASS** `lead-bb1e4165@onb-aca5b34b.test` holds no link to tenant C's IdP
`lead-bb1e4165@onb-aca5b34b.test` (signed in through B's IdP) names tenant C:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <lead-bb1e4165@onb-aca5b34b.test token>
X-Tenant-Id: aca5b34b-183f-5a7c-bd8d-aec2f9261d51
```

```http
HTTP/1.1 401

{"type":"urn:nava:problem:tenant_unresolved","title":"Tenant unresolved","status":401,"code":"tenant_unresolved","detail":"The request could not be tied to a tenant.","instance":"urn:nava:request:0c8b6f5153cfa27f6a70fdf6b43b2115"}
```

- **PASS** `/me` never resolves tenant C (got 401, tenant `00000000-0000-0000-0000-000000000000`)
