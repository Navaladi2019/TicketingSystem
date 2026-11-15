---
type: rfc
id: RFC-018
title: "ADR-007 amendment A1: embedding clients always request organization:<alias>; restated expected behaviour for the S1 deviations"
status: CLOSED (rev 1, final) — decided by ADR-007-A1. S1b disproved rev 1's claim 3: D1-A1.4(b) final text, CI-E14d and the withdrawal of RR-A1-5 are in the ADR, not here.
date: 2026-11-14
issue: TICK-843
amends: "[[ADR-007-embedding-widget-and-sdk]] D1, D13; RFC-017 §7 S1 pass conditions, §8 CI-E2; S23 AC3 and S26 AC5 (TICK-802, TICK-803)"
evidence: docs/adr/ADR-007/spikes/s1-host-client-token.md (TICK-806, PR #652, SPIKE: FAIL); Keycloak 26.8.1 source reading by the Technical Architect (critique round 1); S1b pending
author: Solution Architect
reviewers: Technical Architect (critique round 1, non-blocking), Security & Multi-Tenancy Reviewer (round 1 CHANGES, F1 blocking), CTO (round 1 CHANGES)
---

# RFC-018: ADR-007 A1. Embedding clients always name the organization

> **Invariants source.** The project description defines no invariants (it is empty). As in ADR-007, this RFC is checked against [[ADR-001-platform-blueprint]] §1, I1–I9 (§6).
>
> **Numbering.** RFC-016 is reserved (ADR-006 moved free-text search there); RFC-017 is ADR-007's RFC. This is **RFC-018**; the amendment is **ADR-007-A1**.

## Changelog rev 0 → rev 1

| Source | Change |
|---|---|
| Security F1 (blocking) | D1-A1.3 values are **derived from `Principal.TenantID`** (`onboard.Alias`, `keycloak.TenantIdPAlias`). No read of `tenant_identity_binding`; the false "under RLS" claim is removed. `idp_hints` becomes the single `idp_hint`. CI-E16 a–g taken verbatim. The I3 grant question is gone. |
| Security F2 | `GET /api/v1/embedding-clients/{client_id}` added to D13 with three rules (D1-A1.7). |
| Security F3 + CTO | A `PUT` body carrying `organization_alias` or `idp_hint` gets **400 `unknown_field`**, writes nothing. |
| Security F4 | CI-E14 asserts the invariant (no token names a non-member org); the form is a separate UX check. |
| TA F1, F2 | D1-A1.4 replaced with the three-case text (A form, B claimless token + 401, C `invalid_scope`). CI-E14 split a–d; CI-E2(b) and S23 AC3 reworded. |
| TA F3 | New residual RR-A1-5 (new user created with no org membership). Pairing sentence added to D1-A1.3. |
| TA F6 | RR-A1-1 restated as path A only. |
| TA F7 | §5 option-3 reason strengthened with the source line. |
| CTO | D1-A1.4(b), CI-E14b and CI-E14d marked **source-verified, pending S1b**. S1b is a precondition for ACCEPTED, not for rev 1. |
| Found while revising | The dev seed tenants in `realm-nava.json` use hand-named aliases (`tenant-a`, `idp-tenant-a`), not the derived ones. See §3.1. |

## 1. Decision

**Option 1, made a contract.** Every authorization request from an **embedding client** (any client in `embedding_client_registry`) carries scope `organization:<alias>` for the tenant the host acts for, and SHOULD carry `kc_idp_hint=<idp alias>` of that same tenant. Bare `organization` is **not supported** for embedding clients. The host gets both values as one pair from D13, where the platform derives them from the caller's verified tenant id. `keycloak.Adapter.Resolve` stays byte-identical. The realm's flows do not change. `nava-web` is untouched.

Options 2, 3 and 4 are rejected (§4). Option 4 is not chosen, so no follow-up spike is planned for it. Spike **S1b** confirms the source reading of D1-A1.4(b) at runtime, and it must pass before ACCEPTED.

## 2. The problem in one paragraph

S1 showed that a user in two orgs, on a **first** sign-in through the broker (`kc_idp_hint`), who requests bare `organization`, gets no picker and a token with no `organization` claim. `Resolve` then returns 401 `tenant_unresolved: no organization` (3 of 3 runs, Chromium with third-party cookies blocked and WebKit). The Identity Provider Redirector brokers before the `Organization` authenticator runs, and the picker needs a known user (`OrganizationAuthenticator.java:280`), which a first brokered sign-in does not have yet. Isolation held: no token resolved to a wrong tenant.

## 3. The amended D1 (text for ADR-007-A1)

| # | Rule |
|---|---|
| **D1-A1.1** | An embedding client's authorization request MUST carry `organization:<alias>`, where `<alias>` is the Keycloak organization alias of the tenant the host acts for. The alias stays a **hint, not a trust input**: Keycloak keeps only orgs the user belongs to (`OrganizationScope.java:102`), and the tenant still comes only from the signed `organization.id` through `tenant_identity_binding` (I2). |
| **D1-A1.2** | The host SHOULD also send `kc_idp_hint=<idp alias>` of the **same** tenant. With it, a brokered user with a live IdP session signs in with no prompt. Without it, see D1-A1.5. |
| **D1-A1.3** | **Where the host gets the values.** `GET` and `PUT /api/v1/embedding-clients/{client_id}` return two read-only members: `organization_alias = onboard.Alias(p.TenantID)` (`t-<uuid>`) and `idp_hint = keycloak.TenantIdPAlias(p.TenantID)` (`idp-<uuid>`). `p` is the request's `identity.Principal`, and `TenantID` is set only by `Resolve`. **No table is read** to produce them, so no RLS, policy or grant is involved, and no other tenant's value can appear. `organization_alias` and `idp_hint` come from the same tenant and are used as one pair: the SDK and kit helpers take the D1-A1.3 response object, not two independent strings. A mismatched pair (alias of A, hint of B's IdP) is the only way a correctly configured host reaches RR-A1-5. |
| **D1-A1.4** | **Expected behaviour, alias the user cannot use.** (a) The org exists, the user is not a member, no `kc_idp_hint`: Keycloak shows the realm's sign-in form, even with an SSO session (`CookieAuthenticator.java:90-92`). No code is issued. (b) Same, **with** `kc_idp_hint`: Keycloak brokers, issues a code and a token with **no `organization` claim**, and `Resolve` answers 401 `tenant_unresolved`. If this was the user's first brokered sign-in, Keycloak creates the user **without** org membership (RR-A1-5). *Source-verified (Keycloak 26.8.1, TA critique round 1), pending S1b.* (c) The alias names no enabled org: `error=invalid_scope` to `redirect_uri`, no code (`AuthorizationEndpoint.java:194-200`). In every case no token resolves to a tenant the user is not in. |
| **D1-A1.5** | **Expected behaviour, no `kc_idp_hint`.** When only the upstream IdP session survives and the host sends no `kc_idp_hint`, Keycloak shows its identity-first username form. This is a prompt, accepted as FAIL-UX under ADR-007's S1 rule ("prompt shown → accepted as a UX cost, recorded"). It is not a tenant-resolution failure. |
| **D1-A1.6** | A claimless token (bare `organization` from a multi-org user, or case (b) above) gets 401 `tenant_unresolved` from the unchanged `Resolve`, which requires exactly one org. This is the fail-closed backstop, not a supported path. The SDK documentation and any sign-in helper type make the D1-A1.3 pair required. |
| **D1-A1.7** | **D13 gains `GET /api/v1/embedding-clients/{client_id}`.** (1) Tenant admin only, the same gate as `PUT` (403 otherwise, neither member returned). (2) A `client_id` the caller's tenant has no `embedding_client` row for returns the **same 404** as an unknown `client_id`, byte-identical, with no tenant fields. (3) Audited like `PUT`. **`PUT`** rejects a body that carries `organization_alias` or `idp_hint` with **400 `unknown_field`** (RFC 9457, `errors.yaml`) and writes nothing. |

All other D1 text holds: Keycloak client in the installation realm, public + PKCE or confidential, `nava-api` audience, top-level redirect, no `check-sso` iframe.

### 3.1 The dev seed tenants do not follow the derivation

`realm-nava.json` seeds `tenant-a` / `tenant-b` with hand-named aliases (`tenant-a`, `idp-tenant-a`), while onboarding (`internal/onboard/onboard.go:188`, `onboard.go:220`) creates `t-<uuid>` / `idp-<uuid>`. For a seeded tenant, D1-A1.3 would return `t-00000000-0000-4000-8000-0000000000a1`, which names no org, so the host lands on D1-A1.4(c), `invalid_scope`. That fails closed and visibly, but it would break kits that sign in as the seeded users.

**Decision:** the derivation is the contract, because production tenants exist only through onboarding. The S23 and S26 kits and CI-E14/E15/E16 use **tenants created through the onboarding API** on the fresh stack, not the seeded `tenant-a` / `tenant-b`. Renaming the seed aliases in `realm-nava.json` was rejected for A1: it touches `TestRealmFile`, `stack_test.go`'s alias assertions, the README and the IdP broker endpoints for a fixture-only gain. Recorded as RR-A1-6.

### The sign-in after A1

```mermaid
sequenceDiagram
  participant H as Host page
  participant P as nava API
  participant K as Keycloak (nava realm)
  participant U as Tenant's IdP
  Note over H,P: once, at opt-in: tenant admin GET/PUT /embedding-clients/{id} → {organization_alias, idp_hint}
  H->>K: GET /auth scope="openid organization:t-<A>" kc_idp_hint=idp-<A> (PKCE)
  K->>U: 302 (Identity Provider Redirector; no prompt if IdP session exists)
  U-->>K: 302 broker endpoint
  alt user is a member of A
    K-->>H: 302 /callback?code → token organization = {t-<A>: {id}}
    H->>P: bearer token → Resolve → tenant A → client gate (D2/D3) → 200 or 403
  else user is not a member of A (D1-A1.4 b)
    K-->>H: 302 /callback?code → token with no organization claim
    H->>P: bearer token → Resolve → 401 tenant_unresolved
  end
```

## 4. Options weighed against the S1 evidence

| # | Option | Evidence | Effect on existing sign-in | Verdict |
|---|---|---|---|---|
| **1** | Hosts always send `organization:<alias>` | **Direct PASS** on the failing case: two-org user, first sign-in, `organization:org-b` + hint → tenant B, both engines. Also PASS with SSO and single-org. | None: no realm change; `nava-web` keeps its default bare `organization`. | **Chosen.** A host cannot know before sign-in whether a user is in more than one org, so the rule covers every embedding client. |
| 2 | SDK makes a second top-level redirect on `tenant_unresolved` | Partial: the picker appeared when a Keycloak SSO session existed. The full loop was not run. | None in the realm. | **Rejected.** Contradicts D7 (the widget never runs a login flow), puts sign-in logic in the SDK, needs loop guards, and shows the picker on every first sign-in. |
| 3 | Custom browser flow: `Organization` sub-flow before the redirector | Not tested. Source: the picker returns `false` when `user == null` (`OrganizationAuthenticator.java:280`), so moving the sub-flow earlier cannot show a picker on a first brokered sign-in. | §5. | **Rejected.** It cannot work by construction, and its blast radius is realm-wide. |
| 4 | `kc.org.broker.redirect.mode.email-matches` | Untested. It targets deviation D1-A1.5 (no-hint prompt), not the missing picker. | Would change the identity-first form for every org-linked IdP. | **Rejected, not chosen,** so no spike planned. Revisit under RR-A1-4. |

## 5. Option 3 against the current `realm-nava.json`

| Fact in the export | Consequence of a realm-wide custom browser flow |
|---|---|
| `browserFlow` is unset (built-in `browser`). Custom flows are only `nava-operator-direct-grant`, `nava-operator-direct-grant-role-check`, `nava-first-broker-login`. | Option 3 sets `browserFlow` and changes browser sign-in for **every** browser client in `nava`. |
| `nava-web` has bare `organization` as a **default** scope; the platform's sign-in sends `kc_idp_hint` (`internal/identitystack/stack_test.go` asserts exactly one org key, no prompt). | If the identity-first step ran before the redirector, the hinted platform sign-in could stop at a username form: a regression. Since the user is unknown at that point, it cannot show a picker either: no gain. |
| Both IdPs use `nava-first-broker-login` (`idp-create-user-if-unique`, `idp-add-organization-member`); no post-broker flow. | Untouched by a browser-flow change. |
| `nava-operator-cli` binds `direct_grant`; service clients use `client_credentials`. | Untouched. |

A per-client `authenticationFlowBindingOverrides.browser` would keep `nava-web` out, but the mechanism still cannot produce a picker.

## 6. Invariant check (ADR-001 §1)

| Invariant | Effect |
|---|---|
| I1 tenant wall | D1-A1.3 reads no table. The `embedding_client` row lookup for the GET 404 rule is a tenant table under forced RLS (D2). Holds. |
| I2 tenant from verified source | Alias and hint are derived from `Principal.TenantID`, which only `Resolve` sets; `X-Tenant-Id` and body values are never read (CI-E16 d, e). Alias stays a hint. Holds. |
| I3 roles | No new role, grant or table read. Holds. |
| I4, I5, I6, I9 | Not touched. |
| I7 idempotency | `PUT` replay unchanged; `GET` has no side effect beyond its audit entry, which follows the D13 audit rule. Holds. |
| I8 configuration not code | No realm or trust-path code change. Holds. |

## 7. Tests and kits, rewritten

### RFC-017 §7 S1 pass conditions (restated)

The S1 row is replaced by: *a token from `organization:<alias>` passes unchanged `Resolve` for single-org and two-org users, on a first brokered sign-in and with SSO; an alias the user cannot use never yields a token that resolves to a tenant (D1-A1.4 a–c); with `kc_idp_hint` and a live IdP session, no prompt; without it, the username prompt is recorded FAIL-UX (D1-A1.5).* Against the S1 evidence plus **S1b**, this is a PASS.

### Spike S1b (precondition for ACCEPTED; CTO routes it)

Hypothesis (TA critique): on Keycloak 26.8.1 with `realm-nava.json`, `kc_idp_hint=<B's IdP>` plus `organization:<A's alias>` for an existing tenant-B user gives a 302 to `redirect_uri` carrying `code`, and the exchanged token has no `organization` claim. For a fresh tenant-B user the first sign-in leaves zero memberships. **Pass:** all three, 3/3 runs, Go HTTP client with cookie jar in `internal/identitystack`. **Fail:** form, error, or an `organization` claim. Then D1-A1.4(b), CI-E14b and CI-E14d revert to the rev 0 text ("no token"), and RR-A1-5 is withdrawn. Writing CI-E14b and E14d first is the spike.

### CI-E cases

All cases use onboarded tenants A and B (§3.1); `<A>`/`<B>` are their derived aliases.

| Id | Test | Fails if |
|---|---|---|
| **CI-E2 (restated)** | `kit-host` registered, enabled only for A. (a) B's user signs in with `organization:t-<B>` + `kc_idp_hint=idp-<B>` → token for B → every `/api/v1` call answers **403 `client_not_enabled`**. (b) Same user with `organization:t-<A>` **and `kc_idp_hint=idp-<B>`** → a token is issued with no `organization` claim and every call answers **401 `tenant_unresolved`** (*pending S1b*). The no-hint form case is CI-E14a. | Host usable by a tenant that never opted in; a token resolves for a non-member alias. |
| **CI-E14a** | No hint, existing alias the user is not in, with and without SSO: the sign-in form comes back, no `code`. | A `code` is issued. |
| **CI-E14b** | `kc_idp_hint=idp-<B>`, `organization:t-<A>`, existing B user: a `code` is issued, the token has no `organization` claim, `Resolve` returns 401 `tenant_unresolved`. *Source-verified, pending S1b.* | The token's `organization` claim names an org the user is not a member of. |
| **CI-E14c** | `organization:no-such-org`, with and without the hint: 302 to `redirect_uri` with `error=invalid_scope`, no `code`. | A `code` is issued. |
| **CI-E14d** | Fresh user, first sign-in via `idp-<B>` with `organization:t-<A>`: afterwards the user exists with **zero** org memberships. Pins RR-A1-5 so a Keycloak upgrade that changes it shows up. *Source-verified, pending S1b.* | The user is a member of A (isolation break), or of B (behaviour changed: re-evaluate RR-A1-5). |
| **Invariant across E14a–d** (Security F4) | Every case, with and without hint, with and without SSO, also asserts: **no issued token's `organization` claim names an org the user is not a member of.** The form and error outcomes are asserted separately as UX checks. | Any token names a non-member org. |
| **CI-E15** | Multi-org user (member of A and B; both bindings list the brokering IdP), first sign-in with the hint and `organization:t-<B>` → claim has exactly `t-<B>` → `Resolve` returns tenant B. Same user with bare `organization` → claimless → **401 `tenant_unresolved`** (D1-A1.6). | The chosen path fails, or the backstop stops failing closed. |
| **CI-E16** (Security, verbatim; file under `internal/tenancy/isolation/`, lands with S23) | see table below | |

| CI-E16 | Request | Must hold |
|---|---|---|
| a | A's admin sends `GET` and `PUT` for `kit-host` (enabled for A) | `organization_alias = t-<A>`, `idp_hint = idp-<A>` |
| b | B's admin, same, with B enabled | B's values only, never A's |
| c | B's admin sends `GET kit-host` when **only A** opted in | 404, byte-identical to an unknown `client_id`; no alias in the body |
| d | B's admin sends a `PUT` body with `organization_alias: t-<A>`, `idp_hint: idp-<A>` | **400 `unknown_field`**; nothing written; B's row unchanged |
| e | B's token with forged `X-Tenant-Id: <A>` on `GET` and `PUT` | B's values; the header is ignored |
| f | Shared-IdP fixture: B's binding also lists `idp-<A>` | `idp_hint` for B is `idp-<B>`, never `idp-<A>` (derived, F1) |
| g | A non-admin in A sends `GET` | 403; neither member returned |

The existing `Resolve` unit tests for a claimless token stay as they are.

### Story acceptance criteria to rewrite (Technical Lead)

- **S23 AC3 (TICK-802):** "`agent@tenant-b` signing in through `kit-host` with B's D1-A1.3 pair gets a token, and every call with it answers 403 `client_not_enabled`. With `organization:<A's alias>` **and `kc_idp_hint=<B's IdP>`**, a token is issued with no `organization` claim and every call answers 401 `tenant_unresolved`. Tenant B's PUT cannot change tenant A's row." The kit's PKCE helper takes the D1-A1.3 response object as its input. Kit tenants are created through onboarding (§3.1).
- **S26 AC5 (TICK-803):** "Signed in as tenant B's agent through the sample host configured with B's D1-A1.3 pair, the widget shows the server's `client_not_enabled` error and no data." Add: "The sample host always sends the pair. A run without `kc_idp_hint` shows Keycloak's username form once, which is expected (FAIL-UX), and then continues." No S26 step expects a 401 from a wrong alias without a hint.
- `examples/host-app/` reads the D1-A1.3 pair from configuration, never a bare scope or two hand-typed strings.

## 8. Residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| RR-A1-1 | **Path A only** (non-member alias, no `kc_idp_hint`): a brokered user is stranded on Keycloak's sign-in form, with no token and no way back to the host. With the hint (D1-A1.2) the host gets a claimless token and a 401 it can handle instead. | Technical Lead (host-app docs) | First host support case; then a login-theme "back to app" link. |
| RR-A1-2 | `nava-web` keeps bare `organization`; a multi-org user's first brokered sign-in to the first-party app has the S1 defect (401, fail-closed). No known user is multi-org today. | Solution Architect | The first user who is a member of two orgs. |
| RR-A1-3 | A brokered user resolves in a second org only if that org's binding also lists the brokering IdP (ADR-001 A.1 step 3). Not new. | Solution Architect | A tenant asks to share users across orgs with different IdPs. |
| RR-A1-4 | Without `kc_idp_hint`, a prompt (D1-A1.5). | Technical Lead | A host that cannot send the hint; then option 4 with its own spike. |
| RR-A1-5 | A **new** brokered user whose first sign-in carries an alias they are not in is created with **no org membership** (`IdpAddOrganizationMemberAuthenticator.java:97-104`) and gets 401 on every later sign-in, `nava-web` included, until an admin adds them. Isolation holds: no wrong org is joined. The D1-A1.3 pair makes this reachable only by a host bug. *Pending S1b.* | Solution Architect | The first such case, or before any host goes live. Mitigation candidate: a `postBrokerLoginFlowAlias` running `idp-add-organization-member` on every login; it changes `realm-nava.json` and needs its own spike. |
| RR-A1-6 | The dev seed tenants (`tenant-a`, `tenant-b`) do not follow the derived alias rule, so D1-A1.3 returns aliases that do not exist for them (fails closed with `invalid_scope`). Kits and CI use onboarded tenants. | Technical Lead | A kit or test that needs the seeded tenants as hosts; then rename the seed aliases. |

Each residual becomes an `AR-TICK-843-NN` row in `docs/security/accepted-risks.md` in the ADR PR.

## 9. Rejected alternatives in one line each

- **Option 2 (SDK second redirect):** contradicts D7; login logic in the SDK; loop risk; picker prompt on every first sign-in.
- **Option 3 (custom browser flow):** cannot show a picker for an unknown user; realm-wide blast radius (§5).
- **Option 4 (email-matches redirect):** untested; addresses D1-A1.5, not the missing picker.
- **Reading `idp_aliases` from `tenant_identity_binding` for D1-A1.3:** the table's `binding_lookup` policy is `USING (true)` for `nava_app` (A.3 allow-list), so RLS does not scope it; a missing predicate leaks another tenant's IdP alias and tenant id (Security F1). Derivation reads nothing.
- **Silently ignoring read-only members on `PUT`:** a failure is visible, an ignored field is not (CTO).
- **Renaming the seed aliases in A1:** fixture-only gain, broad test churn (§3.1, RR-A1-6).
- **Server-side refusal of bare-`organization` embedding tokens in the client gate:** the granted scope is likely the literal `organization:<alias>` (TA, inferred from `OrganizationScope.java:279`), so it is implementable. It is not needed: `Resolve` already fails closed on more than one org. Revisit with RR-A1-5.

## 10. Open questions (for the Technical Lead, after rev 1)

1. **Task placement:** do D1-A1.3 and D1-A1.7 (`GET`, the two derived members, `PUT` 400 `unknown_field`) go into TICK-815, or a separate S23 task?
2. **Seeding:** CI-E15 needs a user who is a member of two onboarded orgs, and CI-E14d needs a fresh brokered user. Can the identity-stack test seed both through the admin API?
3. **§3.1:** confirm the S23 and S26 kits can create their tenants through onboarding on a fresh stack.
