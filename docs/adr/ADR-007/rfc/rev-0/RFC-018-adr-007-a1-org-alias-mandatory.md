---
type: rfc
id: RFC-018
title: "ADR-007 amendment A1: embedding clients always request organization:<alias>; restated expected behaviour for the two S1 deviations"
status: DRAFT (rev 0)
date: 2026-11-14
issue: TICK-843
amends: "[[ADR-007-embedding-widget-and-sdk]] D1, RFC-017 §7 S1 pass conditions, RFC-017 §8 CI-E2; S23 AC3 and S26 AC5 (TICK-802, TICK-803)"
evidence: docs/adr/ADR-007/spikes/s1-host-client-token.md (TICK-806, PR #652, SPIKE: FAIL)
author: Solution Architect
reviewers: Technical Architect (critique), Technical Lead, Security & Multi-Tenancy Reviewer (optional; no isolation change)
---

# RFC-018: ADR-007 A1. Embedding clients always name the organization

> **Invariants source.** The project description defines no invariants (it is empty). As in ADR-007, this RFC is checked against [[ADR-001-platform-blueprint]] §1, I1–I9. Treat the invariant check in §6 as complete against that source, not against a project description.
>
> **Numbering.** RFC-016 is reserved (ADR-006 moved free-text search there) and RFC-017 is ADR-007's RFC, so this is **RFC-018**. The amendment id is **ADR-007-A1** (no earlier ADR-007 amendment exists).

## 1. Decision

**Option 1, made a contract.** Every authorization request from an **embedding client** (any client in `embedding_client_registry`) carries scope `organization:<alias>` for the tenant the host is acting for. Bare `organization` is **not supported** for embedding clients. A host that knows the tenant's brokering IdP also sends `kc_idp_hint=<idp alias>`; this is how a sign-in stays prompt-free. `keycloak.Adapter.Resolve` stays byte-identical. Realm flows in `realm-nava.json` do not change. The first-party client `nava-web` is untouched.

Options 2, 3 and 4 are rejected (§4). Option 4 is not chosen, so no follow-up spike is needed.

## 2. The problem in one paragraph

S1 showed that a user in two orgs, on a **first** sign-in through the broker (`kc_idp_hint`), who requests bare `organization`, gets no picker and a token with no `organization` claim, so `Resolve` returns 401 `tenant_unresolved: no organization` (3 of 3 runs, Chromium with third-party cookies blocked and WebKit). In Keycloak's built-in browser flow, which `nava` uses (`browserFlow` is unset in `realm-nava.json`), `Identity Provider Redirector` runs before the `Organization` sub-flow, so the picker is never reached. Isolation held: no token resolved to a wrong tenant. This is a correctness and UX gap, not an isolation gap.

## 3. The amended D1 (text for ADR-007-A1)

| # | Rule |
|---|---|
| **D1-A1.1** | An embedding client's authorization request MUST carry `organization:<alias>`, where `<alias>` is the Keycloak organization alias of the tenant the host acts for. The alias stays a **hint, not a trust input**: Keycloak keeps only orgs the user belongs to, and the tenant still comes only from the signed `organization.id` through `tenant_identity_binding` (I2). |
| **D1-A1.2** | The host SHOULD also send `kc_idp_hint=<idp alias>` (one of the tenant's bound IdP aliases). With it, a brokered user with a live IdP session signs in with no prompt. Without it, a prompt is shown (D1-A1.5). |
| **D1-A1.3** | The host learns both values from the tenant at opt-in. `PUT` and `GET /api/v1/embedding-clients/{client_id}` (D13) return two **read-only, additive** members: `organization_alias` (the calling tenant's org alias) and `idp_hints` (the IdP aliases in the calling tenant's `tenant_identity_binding`). They come from the caller's own tenant row under RLS (I1). They are not secrets and never select a tenant. |
| **D1-A1.4** | **Expected behaviour, wrong alias (deviation 1).** When the alias names an org the user is not a member of, Keycloak issues **no token**. It renders the realm's own username/password form on the Keycloak origin (seen even with a live SSO session) and never redirects to `redirect_uri` with a `code`. `Resolve` is never reached, so there is no 401. For a brokered user this form is a dead end (no local password; the org-linked IdP button is hidden). Accepted residual RR-A1-1. |
| **D1-A1.5** | **Expected behaviour, no `kc_idp_hint` (deviation 2).** When only the upstream IdP session survives and the host sends no `kc_idp_hint`, Keycloak shows its identity-first username form. This is a prompt, accepted as FAIL-UX under ADR-007's S1 rule ("prompt shown → accepted as a UX cost, recorded"). It is not a failure of tenant resolution. |
| **D1-A1.6** | A claimless token (for example bare `organization` from a multi-org user) still gets 401 `tenant_unresolved` from the unchanged `Resolve`. This is the fail-closed backstop, not a supported path. The SDK's documentation and its TypeScript sign-in helper types (if any) make `organizationAlias` required. |

All other D1 text holds: Keycloak client in the installation realm, public + PKCE or confidential, `nava-api` audience, top-level redirect, no `check-sso` iframe.

### The sign-in after A1

```mermaid
sequenceDiagram
  participant H as Host page
  participant K as Keycloak (nava realm)
  participant U as Tenant's IdP
  participant A as nava API
  H->>K: GET /auth  scope="openid organization:tenant-a"  kc_idp_hint=idp-tenant-a  (PKCE)
  alt user is a member of tenant-a
    K->>U: 302 (Identity Provider Redirector; no prompt if IdP session exists)
    U-->>K: 302 broker endpoint
    K-->>H: 302 /callback?code
    H->>K: POST /token
    K-->>H: token, organization = {tenant-a: {id}}
    H->>A: request with bearer token
    A-->>H: Resolve -> tenant A; client gate (D2/D3) -> 200 or 403
  else user is not a member of tenant-a (D1-A1.4)
    K-->>H: Keycloak sign-in form; no code, no token, nothing reaches the API
  end
```

## 4. Options weighed against the S1 evidence

| # | Option | S1 evidence | Effect on existing sign-in | Verdict |
|---|---|---|---|---|
| **1** | Hosts always send `organization:<alias>` | **Direct PASS** for the failing case: two-org user, first sign-in, `organization:org-b`, `kc_idp_hint=upstream` → tenant B, both engines. Also PASS with SSO (`org-b` → B) and single-org (`org-a` → A). | None: no realm change; `nava-web` keeps its default bare `organization` scope. | **Chosen.** Already prescribed by D1 for multi-tenant hosts; A1 extends it to every embedding client because a host cannot know before sign-in whether a user is in more than one org. Cost: deviation 1 becomes reachable by a host bug (RR-A1-1). |
| 2 | On `tenant_unresolved`, SDK does a second top-level redirect once a Keycloak session exists | Partial PASS: the picker appeared and the picked org resolved when a Keycloak SSO session existed. The full "401 → redirect → picker" loop was not run. | None in the realm. | **Rejected.** It contradicts ADR-007 D7 (the widget never runs a login flow; the host supplies the token) and puts sign-in logic into the SDK. It needs loop guards (a second claimless token must not redirect again), and the picker is itself a prompt on every first sign-in. A host that wants a tenant chooser can build it in its own UI and then send `organization:<alias>`. |
| 3 | Custom browser flow: `Organization` sub-flow before `Identity Provider Redirector` | **Not tested.** The mechanism is doubtful: on a first brokered sign-in the user is unknown until the broker returns, and the picker needs a known user, so moving the sub-flow earlier may change nothing. Worth a spike if ever reconsidered. | See §5: realm-wide it changes sign-in for `nava-web` and every browser client; at best no gain, at worst a new username prompt on the platform's own hinted sign-in. | **Rejected.** Unproven fix with a realm-wide blast radius, when option 1 is proven. |
| 4 | Realm option `kc.org.broker.redirect.mode.email-matches` | **Untested.** As far as I know (verify against Keycloak 26.8.1 docs before relying on it) it redirects the identity-first username form to the org's IdP by email domain. That targets deviation 2 (no-hint prompt), not the missing picker on a hinted first sign-in. | Would change the identity-first form for every org-linked IdP. | **Rejected, not chosen,** so no follow-up spike is planned. Revisit as a UX improvement for D1-A1.5 if hosts cannot supply `kc_idp_hint`; that would need its own spike. |

## 5. Option 3 against the current `realm-nava.json`

Checked on `main` (`ed069dd5` + TICK-806 branch, file `deploy/compose/keycloak/import/realm-nava.json`):

| Fact in the export | Consequence of a realm-wide custom browser flow |
|---|---|
| `browserFlow` is unset, so the realm uses Keycloak's built-in `browser`. Custom flows are only `nava-operator-direct-grant`, `nava-operator-direct-grant-role-check`, `nava-first-broker-login`. | Option 3 adds a fourth flow and sets `browserFlow`. It changes the browser sign-in of **every** browser client in `nava`, not only host apps. |
| `nava-web` (public, first-party) has `organization` as a **default** scope (bare), plus `nava-idp`, `nava-api-audience`. The platform's sign-in sends `kc_idp_hint` (`internal/identitystack/stack_test.go` asserts exactly one org key and no prompt). | If the identity-first `Organization` step runs before the redirector, the hinted platform sign-in can stop at a username form. That is a regression of a passing test and of the product's sign-in. If it does not run (user not yet known), option 3 fixes nothing. |
| Both IdPs (`idp-tenant-a`, `idp-tenant-b`) use `firstBrokerLoginFlowAlias = nava-first-broker-login` (`idp-create-user-if-unique`, `idp-add-organization-member`); no post-broker flow. | Untouched by a browser-flow change; membership on first login keeps working. |
| `nava-operator-cli` binds `direct_grant` to `nava-operator-direct-grant`; service clients use `client_credentials`. | Untouched (not the browser flow). |
| Org IdP links set only `kc.org.domain`; no redirect-by-email mode. All seeded users are single-org. | Today's seed would not exercise the multi-org path at all, so a regression would show only in the existing single-org tests. |

The blast radius could be narrowed with `authenticationFlowBindingOverrides.browser` on host clients only, leaving `nava-web` on the built-in flow. That still leaves the unproven mechanism, so the verdict stands.

## 6. Invariant check (ADR-001 §1)

| Invariant | Effect |
|---|---|
| I1 tenant wall | D1-A1.3 reads only the caller's own tenant row under forced RLS. Holds. |
| I2 tenant from verified source | Alias stays a hint; tenant from signed `organization.id` via `tenant_identity_binding`; `Resolve` unchanged. Holds. |
| I3 roles | No new role or grant. If `idp_hints` needs a read of `tenant_identity_binding`, it must use an existing grant; the Technical Lead confirms in S23 planning. Holds (to confirm). |
| I4, I5, I6, I9 | Not touched. |
| I7 idempotency | `PUT` replay semantics unchanged; two read-only members added. Holds. |
| I8 configuration not code | No realm or code change in the trust path; host behaviour is configuration (alias, hint). Holds. |

## 7. Tests and kits, rewritten

### RFC-017 §7 S1 pass conditions (restated, for the record)

The S1 row is replaced by: *token from `organization:<alias>` passes unchanged `Resolve` (single-org and two-org users, first brokered sign-in and SSO); wrong alias issues no token (D1-A1.4); with `kc_idp_hint` and a live IdP session, no prompt; without `kc_idp_hint`, the username prompt is recorded FAIL-UX (D1-A1.5).* Against the existing evidence this is a **PASS** with no new spike run.

### CI-E cases

| Id | Test | Fails if |
|---|---|---|
| **CI-E2 (restated)** | `kit-host` registered, enabled only for A. (a) `agent@tenant-b` signs in through `kit-host` with `organization:tenant-b` → token for tenant B → any `/api/v1` call answers **403 `client_not_enabled`**. (b) Same user with `organization:tenant-a` → **no authorization code** reaches `redirect_uri`; the Keycloak sign-in form is returned (CI-E14). | Host usable by a tenant that never opted in; or a token issued for a non-member alias. |
| **CI-E14 (new)** | Wrong alias, deviation 1: with and without a live Keycloak SSO session, `GET /auth` with `organization:<alias of an org the user is not in>` answers with the realm's sign-in form (HTML on the Keycloak origin), never a `302` to `redirect_uri` carrying `code`. Runs in `internal/identitystack` with a Go HTTP client and cookie jar, no browser. | A code or token is issued for a non-member alias. |
| **CI-E15 (new)** | Multi-org first brokered sign-in: seed one user in `tenant-a` and `tenant-b` (both bindings list the brokering IdP), first sign-in with `kc_idp_hint` and `organization:tenant-b` → token `organization` has exactly `tenant-b` → `Resolve` returns tenant B. Same user with bare `organization` → claimless token → **401 `tenant_unresolved`** (D1-A1.6 backstop, documented, not supported). | The chosen path fails, or the backstop stops failing closed. |
| **CI-E16 (new)** | D1-A1.3: `GET`/`PUT /api/v1/embedding-clients/{client_id}` as tenant A's admin returns `organization_alias = tenant-a` and `idp_hints = [idp-tenant-a]`; as tenant B's admin, B's values; never the other tenant's. | Another tenant's alias or IdP leaks. |
| Deviation 2 | No CI case asserts "no prompt" without `kc_idp_hint`. The S26 kit records the prompt as expected (below). | n/a |

The existing `Resolve` unit tests for a claimless token (401 `tenant_unresolved`) stay as they are.

### Story acceptance criteria to rewrite (Technical Lead)

- **S23 AC3 (TICK-802):** "`agent@tenant-b` signing in through `kit-host` with `organization:tenant-b` gets a token, and every call with it answers 403 `client_not_enabled`. With `organization:tenant-a` no token is issued and the kit shows the Keycloak sign-in form. Tenant B's PUT cannot change tenant A's row." The kit's PKCE helper takes the alias (and IdP hint) as required inputs.
- **S26 AC5 (TICK-803):** "Signed in as `agent@tenant-b` through the sample host configured for `organization:tenant-b`, the widget shows the server's `client_not_enabled` error and no data." Add: "The sample host always sends `organization:<alias>` and `kc_idp_hint`; a run without `kc_idp_hint` shows Keycloak's username form once, which is expected (FAIL-UX), and then continues." No S26 step expects a 401 from a wrong alias.
- `examples/host-app/` reads `organization_alias` and `idp_hints` from configuration (filled from the D1-A1.3 response), not from a hard-coded bare scope.

## 8. Residuals

| # | Residual | Owner | Revisit trigger |
|---|---|---|---|
| RR-A1-1 | A host that sends a non-member alias strands a brokered user on Keycloak's sign-in form (no token, no way back to the host). | Technical Lead (host-app docs) | First host support case; then spike `prompt=none` (expect `error=login_required` back to the host) or a login-theme "back to app" link. |
| RR-A1-2 | `nava-web` keeps bare `organization`; a multi-org user's first brokered sign-in to the first-party app has the same S1 defect (401, fail-closed). No seeded or known production user is multi-org today. | Solution Architect | The first user who is a member of two orgs, or a story that adds one. |
| RR-A1-3 | A brokered user resolves in a second org only if that org's binding also lists the brokering IdP (ADR-001 A.1 step 3, seen in the S1 fixture). Not new; limits which multi-org users a host can serve. | Solution Architect | A tenant asks to share users across orgs with different IdPs. |
| RR-A1-4 | Without `kc_idp_hint`, a prompt (deviation 2). | Technical Lead | A host that cannot obtain the hint; then option 4 with its own spike. |

## 9. Rejected alternatives in one line each

- **Option 2 (SDK second redirect):** contradicts D7; login logic in the SDK; loop risk; picker prompt on every first sign-in.
- **Option 3 (custom browser flow):** unproven mechanism, realm-wide blast radius on `nava-web` (§5).
- **Option 4 (email-matches redirect):** untested; addresses deviation 2, not the missing picker.
- **Server-side refusal of bare-`organization` embedding tokens in the client gate:** would make D1-A1.1 enforceable, but depends on the exact `scope` claim text for `organization:<alias>`, which S1 did not record. Fail-closed `Resolve` already covers the harmful case (multi-org → 401). Revisit if a single-org host ships with bare scope and breaks later.

## 10. Open questions for reviewers

1. **Technical Architect:** confirm from Keycloak 26.8.1 source that `organization:<alias>` for a non-member always ends at the sign-in form (not an error redirect) regardless of `kc_idp_hint`. S1 tested the wrong alias without the hint only. If the hint changes the outcome, CI-E14 states both.
2. **Technical Lead:** D1-A1.3 adds two read-only members to the D13 responses. Is this in S23 scope (P8 task for TICK-815), or should it be a separate small task? It is the one contract change in A1.
3. **Technical Lead:** CI-E15 needs a multi-org seed user in the CI realm only (not in `realm-nava.json`'s dev seed). Confirm the identity-stack test can seed it through the admin API.
