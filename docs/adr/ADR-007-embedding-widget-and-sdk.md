---
type: adr
id: ADR-007
title: Embedding the platform in other applications — realm-client identity, per-tenant host-app opt-in, web-component widget, TypeScript SDK
status: ACCEPTED
date: 2026-11-14
accepted: 2026-11-14
issue: TICK-792
rfc: RFC-017 (rev 1, docs/adr/ADR-007/rfc/RFC-017-embedding.md; critique docs/adr/ADR-007/rfc/critique-round-1-technical-architect.md)
supersedes: none. Amends ADR-004 R1 (audit envelope v2, D5 below).
---

# ADR-007: Embedding the platform in other applications

**Status: ACCEPTED** on 2026-11-14 by the Solution Architect (merge policy set by vknavaladi). Technical Architect critique round 1 had no blocking findings, so no CTO tie-break was needed. Verdict on TICK-792, PR #644.

> **Invariants source.** The project description defines no invariants; this ADR is checked against [[ADR-001-platform-blueprint]] §1, I1–I9.

## Decision

Other applications ("host apps") embed the platform through **ordinary Keycloak realm tokens** that pass the **unchanged** D26 resolver, so the tenant still comes only from `(iss, organization.id)` (I2). A tenant opts in to each host app and its browser origins; the host app is identified by the verified token's `azp`, which can refuse a request but never chooses a tenant. The UI widget v1 is a **web component** that uses a token the host supplies. The headless SDK v1 is **TypeScript**, generated from the OpenAPI contract. No token exchange, no "on behalf of" header, no iframe in v1.

| # | Decision |
|---|---|
| **D1** | **User-present identity.** A host app is a Keycloak client in the installation realm (public + PKCE for browser-only hosts, confidential for hosts with a backend), with the `nava-api` audience and scope `organization`. A host serving many tenants sends `organization:<alias>`; that alias is a hint, and Keycloak keeps only orgs the user belongs to. Sign-in is a top-level redirect (no prompt when the brokered IdP session exists); no hidden-iframe `check-sso`. `keycloak.Adapter.Resolve` stays byte-identical. |
| **D2** | **Host-app registry and tenant opt-in.** The operator registers a host app: it creates the Keycloak client (D26: only platform service accounts administer Keycloak) and a row in the platform table `embedding_client_registry(client_id, display_name, state)`. A tenant admin enables it in the tenant table `embedding_client(tenant_id, client_id, allowed_origins text[], state, rate_limit_share, created_by, created_at)`, `ENABLE`+`FORCE` RLS, `UNIQUE (tenant_id, client_id)`, `client_id` must exist in the registry. `identity.Principal` gains `ClientID` (from `azp`). A client-gate middleware after auth, inside `InTenantTx`, requires `ClientID` ∈ `NAVA_FIRST_PARTY_CLIENTS` or an `enabled` row, else **403 `client_not_enabled`**. No cache. |
| **D3** | **CORS and origins.** CORS is the outermost middleware, on every response including 401/403/429. Preflight echoes `Origin`; `Allow-Methods: GET, POST, PUT, PATCH, DELETE`; `Allow-Headers: Authorization, Content-Type, Idempotency-Key, If-Match, Nava-SDK`; `Max-Age: 7200`; no `Allow-Credentials`. All responses: `Allow-Origin: <echoed>`, `Vary: Origin`, `Expose-Headers: ETag, Location, Retry-After`. After auth, an embedding client's request with an `Origin` not in `allowed_origins` gets **403 `origin_not_allowed`**. |
| **D4** | **Machine-to-machine.** `client_credentials` on a confidential client whose service-account user is a member of exactly one organization, with a `principals` row `kind = integration` (directory sync records `LinksKnown = true`, no links). It acts only as itself under normal ACLs and masks. **If S2 fails:** an operator-only hardcoded-claim mapper emits the organization, and `Resolve` gains one branch for `kind = integration` principals with no links. |
| **D5** | **Audit attribution (amends ADR-004 R1).** New envelope member `via_client_id` (the embedding client's `azp`, else NULL) and `envelope_version`. v1 = today's bytes. v2 = both members inside the JCS bytes hashed into `envelope_sha256`, NULL emitted. The offline verifier accepts both; new v2 golden vectors; Technical Lead review per PLAN-005 D-5-7. "User Y via host app X" = `actor_principal_id = Y`, `via_client_id = X`. |
| **D6** | **Rate limits.** Per-tenant bucket kept. Per-`(tenant, ClientID)` sub-bucket, rate and burst = `rate_limit_share` (default 50 %) of the tenant's. Both reserved atomically under one lock, both cancelled if either delays. `Limiter.Allow(ctx, Key{Tenant, ClientID})`. |
| **D7** | **Widget v1** `<nava-records entity view="list\|record\|create">`: npm package and a platform-served single-file ES module with SRI (no dynamic `import()`, CI fails on a second chunk). Host passes `getToken()`; the widget calls it per request and once more on 401, never logs in, never stores the token. Theming via CSS custom properties and `::part()`. |
| **D8** | **Widget renders server truth.** Forms from the published JSON Schema; no `eval`/`new Function`; styles via `adoptedStyleSheets`; portals into the shadow root. v1 field kinds: `string` (`date`, `date-time`, `email`, `textarea` hint), `number`, `integer`, `boolean`, single-select (`enum`, `oneOf` of `const`), one nested `object` level. Rules: SHOW/HIDE on `const`/`enum` only, by a small interpreter. Validation and masks are the server's (I4, I5). |
| **D9** | **Widget v1 scope:** list (filters on `x-nava-filterable` fields), view, create with a per-submission `Idempotency-Key` (I7). |
| **D10** | **SDK v1 = TypeScript** (browser, Node 20+), generated from `api/openapi.bundled.yaml` by a pinned generator plus a thin hand-written layer (token provider, `Idempotency-Key`, `If-Match` with typed `VersionConflict`, cursors, RFC 9457 errors by `code`, `Retry-After`). CI regenerates and fails on drift. |
| **D11** | **Versioning.** SDK major = API major; minor = additive; patch = SDK fix. `Nava-SDK: ts/<version>` is diagnostic only, never trusted. |
| **D12** | **Second SDK language** when the first host that cannot use TypeScript or plain REST signs up (candidates: Java, C#, Python, Go). |
| **D13** | **Operations.** `POST /api/v1/operator/embedding-clients` (operator; Keycloak client + registry row), `PUT /api/v1/embedding-clients/{client_id}` (tenant admin; origins, state, share). Both audited. |

## Context it rests on

D26 tenant resolution, fail-closed (ADR-001 A.1); REST-first `/api/v1` from OpenAPI 3.1, additive (D6); per-tenant limiter (T24); audit envelope (ADR-004 R1). Keycloak 26.8.1 `OrganizationScope` and `addMember` behaviour read from source by the Technical Architect (critique F2, F10).

## Consequences

- Host apps need operator registration plus tenant opt-in; there is no self-service Keycloak client creation (R3 holds).
- The audit export gains envelope v2; verifiers must accept both forms.
- `ratelimit.Limiter` changes signature; every caller is updated in one task.
- The widget supports only the D8 field kinds; other fields are read-only or block `create`.

## Spikes that gate the plan (pass/fail fixed in RFC-017 rev 1 §7)

| Spike | Gates | If it fails |
|---|---|---|
| S1 Keycloak 26.8.1 host-client token passes unchanged `Resolve`; no prompt on top-level redirect (WebKit, Chromium without third-party cookies) | D1 | Token refused → amendment to this ADR. Prompt shown → accepted as a UX cost, recorded. |
| S2 service account as org member, sync row, `Resolve` | D4 | D4 fallback (already decided). |
| S3 renderer in shadow root under strict CSP | D8 | JSONForms with custom rule evaluator and `adoptedStyleSheets`; then a hand-written renderer. |
| S4 CORS for 200/401/403/429 in three engines; p95 of the D2 lookup | D3, D2 budget | Fix middleware order; the measured p95 becomes the D2 budget. |

## API test scenarios

| Operation | Scenarios |
|---|---|
| `POST /api/v1/operator/embedding-clients` | 201; 400 invalid body writes nothing (no Keycloak client, no row); 401 non-operator token; 409 duplicate `client_id`; idempotent replay creates one client and one row; audit entry written. |
| `PUT /api/v1/embedding-clients/{client_id}` | 200 create/update; 400 invalid origin (wildcard, path, non-https outside dev) writes nothing; 403 non-admin; 404 client not in registry; 412 stale `If-Match`; replay writes once; tenant B's PUT never changes tenant A's row; audit entry written. |
| Every `/api/v1` operation (client gate) | 403 `client_not_enabled` for a disabled or never-enabled embedding client (next request after disable); 403 `origin_not_allowed`; first-party client unaffected; CORS headers on 401/403/429; forged `X-Tenant-Id`, `X-On-Behalf-Of`, `Nava-SDK` change nothing; integration principal refused on every CI-2 cross-tenant path. |

Full test list: RFC-017 §8, CI-E1 to CI-E13.

## Residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| RR1 | Host-page JavaScript can read the user's 5-minute, `nava-api`-only token. | Security reviewer | A host with untrusted third-party scripts on the embedding page → iframe option. |
| RR2 | A leaked M2M client secret acts as that integration principal in its one tenant. | Technical Lead | Any request for an integration principal spanning tenants (refused). |
| RR3 | No origin check without an `Origin` header (server callers). | Solution Architect | Evidence of token replay from servers. |
| RR4 | Sub-buckets are per replica. | Technical Lead | Shared limiter store lands. |

## Rejected options

| Option | Why |
|---|---|
| RFC 8693 token exchange from the host's IdP token | A second way to mint user tokens outside the hardened first-broker-login flow. Revisit: a host whose users are in no IdP the tenant has bound. |
| Trusted host backend + `X-On-Behalf-Of` header | A client value chooses the actor; one leaked secret acts as every user. |
| Host-signed user assertions (RFC 7523) | Makes every host an identity provider for our tenants (I2). |
| Iframe widget, alone or alongside | Third-party-cookie blocking breaks in-frame sign-in; needs a postMessage token channel and per-tenant `frame-ancestors` served before the tenant is known. Revisit with RR1. |
| Hand-written SDKs | Drift from the contract. |
| CORS allow-list on preflight | Preflight carries no token, so the check needs a cross-tenant read (I1, I3). |
| Overloading `source_ref` for the host app | `source_ref` is reserved for AI decisions and change sets (ADR-004 R4). |
| GraphQL for the widget | Epic 23; one authorization surface (D6). |

**Dissent: none recorded.** Technical Architect critique round 1 had no blocking findings; all non-blocking items were accepted in rev 1. No CTO tie-break was needed. The Security & Multi-Tenancy Reviewer was not asked for a separate review; no tenant-isolation gap was found.

## Out of scope for v1

Iframe; token exchange; widget update, delete, workflow actions, comments, attachments, audit view; GraphQL; SDK languages other than TypeScript; tenant self-service Keycloak clients; offline use; per-user rate limits; real-time push.
