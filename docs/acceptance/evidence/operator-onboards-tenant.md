# S4 acceptance evidence (TICK-46)

Recorded by `TestS4Acceptance` (`internal/identitystack/s4_acceptance_test.go`) on 2026-10-27T20:26:19Z. Tokens are never recorded; the IdP client secret is redacted.

Tenant A: `dfce860c-7452-5ea4-86c3-13f1141ff588`, domain `onb-dfce860c.test`, admin `admin@onb-dfce860c.test`.
Tenant B: `d95b87ed-b37c-5af6-984e-a87e63ca118e`, domain `onb-d95b87ed.test`, admin `admin@onb-d95b87ed.test`.


## Criterion 1: 201 with the tenant id; a replay returns the same body and creates nothing

First call, tenant onb-dfce860c.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 674d750e-37a6-4280-8b1a-b302ca6617a4
Content-Type: application/json

{
  "admin_email": "admin@onb-dfce860c.test",
  "email_domains": [
    "onb-dfce860c.test"
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
    "display_name": "Onboarded onb-dfce860c.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"dfce860c-7452-5ea4-86c3-13f1141ff588"}
```

- **PASS** first call: 201, tenant_id `dfce860c-7452-5ea4-86c3-13f1141ff588`, state `active` (want 201, `dfce860c-7452-5ea4-86c3-13f1141ff588`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 674d750e-37a6-4280-8b1a-b302ca6617a4
Content-Type: application/json

{
  "admin_email": "admin@onb-dfce860c.test",
  "email_domains": [
    "onb-dfce860c.test"
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
    "display_name": "Onboarded onb-dfce860c.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"dfce860c-7452-5ea4-86c3-13f1141ff588"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-dfce860c-7452-5ea4-86c3-13f1141ff588`, after `1 tenants, 1 bindings, 1 email domains, 1 invites, 1 Keycloak organizations t-dfce860c-7452-5ea4-86c3-13f1141ff588`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 674d750e-37a6-4280-8b1a-b302ca6617a4
Content-Type: application/json

{
  "admin_email": "admin@onb-dfce860c.test",
  "email_domains": [
    "onb-dfce860c.test"
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
    "display_name": "Changed onb-dfce860c.test",
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
- **PASS** the admin `admin@onb-dfce860c.test` has a pending invite and no Keycloak user yet
First call, tenant onb-d95b87ed.test:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 3f3a4df7-5c1c-4279-847d-b0aa2412ec6a
Content-Type: application/json

{
  "admin_email": "admin@onb-d95b87ed.test",
  "email_domains": [
    "onb-d95b87ed.test"
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
    "display_name": "Onboarded onb-d95b87ed.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"d95b87ed-b37c-5af6-984e-a87e63ca118e"}
```

- **PASS** first call: 201, tenant_id `d95b87ed-b37c-5af6-984e-a87e63ca118e`, state `active` (want 201, `d95b87ed-b37c-5af6-984e-a87e63ca118e`, `active`)
Replay, same key and body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 3f3a4df7-5c1c-4279-847d-b0aa2412ec6a
Content-Type: application/json

{
  "admin_email": "admin@onb-d95b87ed.test",
  "email_domains": [
    "onb-d95b87ed.test"
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
    "display_name": "Onboarded onb-d95b87ed.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 200

{"state":"active","tenant_id":"d95b87ed-b37c-5af6-984e-a87e63ca118e"}
```

- **PASS** replay status 200 (documented: 200 for a replay of a completed request)
- **PASS** replay body is byte-identical to the first body
- **PASS** nothing new: before replay `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-d95b87ed-b37c-5af6-984e-a87e63ca118e`, after `2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-d95b87ed-b37c-5af6-984e-a87e63ca118e`
Same key, different body:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 3f3a4df7-5c1c-4279-847d-b0aa2412ec6a
Content-Type: application/json

{
  "admin_email": "admin@onb-d95b87ed.test",
  "email_domains": [
    "onb-d95b87ed.test"
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
    "display_name": "Changed onb-d95b87ed.test",
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
- **PASS** the admin `admin@onb-d95b87ed.test` has a pending invite and no Keycloak user yet

## Criterion 2: the new admin signs in and is scoped to the new tenant

- **PASS** `admin@onb-dfce860c.test` signed in through IdP `idp-dfce860c-7452-5ea4-86c3-13f1141ff588`; the token's `organization` claim is exactly `t-dfce860c-7452-5ea4-86c3-13f1141ff588`
- **PASS** `admin@onb-d95b87ed.test` signed in through IdP `idp-d95b87ed-b37c-5af6-984e-a87e63ca118e`; the token's `organization` claim is exactly `t-d95b87ed-b37c-5af6-984e-a87e63ca118e`
One `nava-groupsync --once` cycle ran (the long-running service is paused for the test so its own cycle cannot hold a tenant's lock).

Admin of onb-dfce860c.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-dfce860c.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-dfce860c-7452-5ea4-86c3-13f1141ff588","tenant_id":"dfce860c-7452-5ea4-86c3-13f1141ff588","user_id":"185cb921-6a6c-45be-90c4-590fc3f2f010"}
```

- **PASS** `/me` is tenant `dfce860c-7452-5ea4-86c3-13f1141ff588` via `idp-dfce860c-7452-5ea4-86c3-13f1141ff588` (got 200, tenant `dfce860c-7452-5ea4-86c3-13f1141ff588`, idp `idp-dfce860c-7452-5ea4-86c3-13f1141ff588`)
- **PASS** `admin@onb-dfce860c.test` is in `/t/dfce860c-7452-5ea4-86c3-13f1141ff588/` and the invite is consumed
Admin of onb-d95b87ed.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-d95b87ed.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-d95b87ed-b37c-5af6-984e-a87e63ca118e","tenant_id":"d95b87ed-b37c-5af6-984e-a87e63ca118e","user_id":"d4efebe6-a61b-4088-9c9d-15330487d335"}
```

- **PASS** `/me` is tenant `d95b87ed-b37c-5af6-984e-a87e63ca118e` via `idp-d95b87ed-b37c-5af6-984e-a87e63ca118e` (got 200, tenant `d95b87ed-b37c-5af6-984e-a87e63ca118e`, idp `idp-d95b87ed-b37c-5af6-984e-a87e63ca118e`)
- **PASS** `admin@onb-d95b87ed.test` is in `/t/d95b87ed-b37c-5af6-984e-a87e63ca118e/` and the invite is consumed

## Isolation check: tenant B cannot see or change what onboarding created for tenant A

Admin of onb-d95b87ed.test names tenant onb-dfce860c.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-d95b87ed.test token>
X-Tenant-Id: dfce860c-7452-5ea4-86c3-13f1141ff588
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-d95b87ed-b37c-5af6-984e-a87e63ca118e","tenant_id":"d95b87ed-b37c-5af6-984e-a87e63ca118e","user_id":"d4efebe6-a61b-4088-9c9d-15330487d335"}
```

- **PASS** still tenant `d95b87ed-b37c-5af6-984e-a87e63ca118e`, never `dfce860c-7452-5ea4-86c3-13f1141ff588`
Admin of onb-d95b87ed.test names tenant onb-dfce860c.test:

```http
GET /api/v1/me?tenant_id=dfce860c-7452-5ea4-86c3-13f1141ff588 (nava-core)
Authorization: Bearer <admin@onb-d95b87ed.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-d95b87ed-b37c-5af6-984e-a87e63ca118e","tenant_id":"d95b87ed-b37c-5af6-984e-a87e63ca118e","user_id":"d4efebe6-a61b-4088-9c9d-15330487d335"}
```

- **PASS** still tenant `d95b87ed-b37c-5af6-984e-a87e63ca118e`, never `dfce860c-7452-5ea4-86c3-13f1141ff588`
- **PASS** `admin@onb-d95b87ed.test` is not in `/t/dfce860c-7452-5ea4-86c3-13f1141ff588/` and not a member of organization `t-dfce860c-7452-5ea4-86c3-13f1141ff588`
- **PASS** `admin@onb-d95b87ed.test` holds no link to `idp-dfce860c-7452-5ea4-86c3-13f1141ff588`
Admin of onb-dfce860c.test names tenant onb-d95b87ed.test:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <admin@onb-dfce860c.test token>
X-Tenant-Id: d95b87ed-b37c-5af6-984e-a87e63ca118e
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-dfce860c-7452-5ea4-86c3-13f1141ff588","tenant_id":"dfce860c-7452-5ea4-86c3-13f1141ff588","user_id":"185cb921-6a6c-45be-90c4-590fc3f2f010"}
```

- **PASS** still tenant `dfce860c-7452-5ea4-86c3-13f1141ff588`, never `d95b87ed-b37c-5af6-984e-a87e63ca118e`
Admin of onb-dfce860c.test names tenant onb-d95b87ed.test:

```http
GET /api/v1/me?tenant_id=d95b87ed-b37c-5af6-984e-a87e63ca118e (nava-core)
Authorization: Bearer <admin@onb-dfce860c.test token>
```

```http
HTTP/1.1 200

{"groups":[],"idp_alias":"idp-dfce860c-7452-5ea4-86c3-13f1141ff588","tenant_id":"dfce860c-7452-5ea4-86c3-13f1141ff588","user_id":"185cb921-6a6c-45be-90c4-590fc3f2f010"}
```

- **PASS** still tenant `dfce860c-7452-5ea4-86c3-13f1141ff588`, never `d95b87ed-b37c-5af6-984e-a87e63ca118e`
- **PASS** `admin@onb-dfce860c.test` is not in `/t/d95b87ed-b37c-5af6-984e-a87e63ca118e/` and not a member of organization `t-d95b87ed-b37c-5af6-984e-a87e63ca118e`
- **PASS** `admin@onb-dfce860c.test` holds no link to `idp-d95b87ed-b37c-5af6-984e-a87e63ca118e`
A new tenant claims tenant A's domain:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: 0993ff07-cc96-4837-a387-6648d843c245
Content-Type: application/json

{
  "admin_email": "admin@onb-dfce860c.test",
  "email_domains": [
    "onb-dfce860c.test"
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
    "display_name": "Onboarded onb-dfce860c.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 409

{"type":"urn:nava:problem:domain_claimed","title":"Email domain already claimed","status":409,"code":"domain_claimed","detail":"An email domain of the request is already claimed by a tenant.","instance":"urn:nava:request:50c2d1dc88a32b5af226f438c02a99ec"}
```

- **PASS** 409 `domain_claimed` (want 409 `domain_claimed`)
- **PASS** the body names neither tenant A (`dfce860c-7452-5ea4-86c3-13f1141ff588`) nor its organization (`t-dfce860c-7452-5ea4-86c3-13f1141ff588`, id `c1150cd5-5a92-4ade-9db3-379415d29aa9`) nor its IdP
- **PASS** `onb-dfce860c.test` is still owned by tenant A (got `dfce860c-7452-5ea4-86c3-13f1141ff588`)
- **PASS** no `tenants` row (provisioning or otherwise) for the refused tenant `e0026046-1356-56ab-bd91-08f5a7ab409b` (got 0)
- **PASS** no Keycloak organization `t-e0026046-1356-56ab-bd91-08f5a7ab409b` (got [])
- **PASS** no second invite for `admin@onb-dfce860c.test`
Tenant B's admin replays A's onboarding on nava-core:

```http
POST /api/v1/operator/tenants (nava-core)
Authorization: Bearer <admin@onb-d95b87ed.test token>
Idempotency-Key: 674d750e-37a6-4280-8b1a-b302ca6617a4
```

```http
HTTP/1.1 501

{"type":"urn:nava:problem:not_implemented","title":"Not implemented","status":501,"code":"not_implemented","detail":"This operation is declared in the contract but not implemented yet.","instance":"urn:nava:request:729b4cff769496626cf4aa2bf8f3de52"}
```

- **PASS** refused with 501; nothing changes for tenant A (`2 tenants, 2 bindings, 2 email domains, 2 invites, 1 Keycloak organizations t-dfce860c-7452-5ea4-86c3-13f1141ff588`)

## Criterion 3: inviting a user already federated to another tenant's IdP is rejected

- **PASS** precondition: `lead-4b3749a7@onb-088157bb.test` exists and is federated to tenant B's IdP only (`[idp-d95b87ed-b37c-5af6-984e-a87e63ca118e]`)
Onboard tenant C naming that user as admin:

```http
POST /api/v1/operator/tenants (nava-onboard :8081)
Authorization: Bearer <operator@nava.test token>
Idempotency-Key: f9883784-2f5a-4285-8456-d08337b15d49
Content-Type: application/json

{
  "admin_email": "lead-4b3749a7@onb-088157bb.test",
  "email_domains": [
    "onb-088157bb.test"
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
    "display_name": "Onboarded onb-088157bb.test",
    "provider_id": "oidc"
  }
}
```

```http
HTTP/1.1 201

{"state":"active","tenant_id":"088157bb-27c5-55cb-9afd-bce4754002df"}
```

Status 201. By design (`docs/operator/onboarding.md` step 5) a refused invite is answered exactly like a sent one.

- **PASS** no invite for `lead-4b3749a7@onb-088157bb.test` in tenant C (got `missing`)
- **PASS** signing in through tenant C's IdP as `lead-4b3749a7@onb-088157bb.test` is refused because the account already exists (Keycloak says: "efox = true; nava We are sorry... User with email lead-4b3749a7@onb-088157bb.test already exists. Please login to account management to link the accou")
- **PASS** control: `admin@onb-088157bb.test` signs in through the same IdP of tenant C (no error text on the page)
- **PASS** after a group sync cycle `lead-4b3749a7@onb-088157bb.test` is not in `/t/088157bb-27c5-55cb-9afd-bce4754002df/` and not a member of `t-088157bb-27c5-55cb-9afd-bce4754002df`
- **PASS** `lead-4b3749a7@onb-088157bb.test` holds no link to tenant C's IdP
`lead-4b3749a7@onb-088157bb.test` (signed in through B's IdP) names tenant C:

```http
GET /api/v1/me (nava-core)
Authorization: Bearer <lead-4b3749a7@onb-088157bb.test token>
X-Tenant-Id: 088157bb-27c5-55cb-9afd-bce4754002df
```

```http
HTTP/1.1 401

{"type":"urn:nava:problem:tenant_unresolved","title":"Tenant unresolved","status":401,"code":"tenant_unresolved","detail":"The request could not be tied to a tenant.","instance":"urn:nava:request:cf797b49e763affcc3ed19b6e0ecafd6"}
```

- **PASS** `/me` never resolves tenant C (got 401, tenant `00000000-0000-0000-0000-000000000000`)
