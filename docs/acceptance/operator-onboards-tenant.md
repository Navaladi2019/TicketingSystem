# Acceptance kit: S4, an operator onboards a tenant and its admin signs in

Story: TICK-46. Plan: PLAN-001 (`docs/delivery/plan-platform-foundation.md`). Design: ADR-001. Operator reference: [`docs/operator/onboarding.md`](../operator/onboarding.md).

The Quality, Testing & CI Advisor verified this on 2026-10-28, on `TICK-46/operator-onboards-tenant` at `299bdff`: current `main` merged in at `8ef35f2`, including TICK-160, which pins Keycloak's issuer, then the TICK-278 fix (PR #217), which makes `nava-onboard` trust that pinned issuer. The run itself was on PR #217's head `f38301c`, whose tree is identical to `299bdff`. The fresh stack came up with `docker compose down -v`, then `docker compose build && docker compose up --wait`, with the "Try it" ports below. Every service was healthy. `TestS4Acceptance` passed 42 of 42 checks. CI: PR #217 merged with all 8 jobs green ([40584903710](https://github.com/Navaladi2019/TicketingSystem/actions/runs/40584903710)).

Earlier runs: 42 of 42 at `d256e86`, before the `main` merge. The first run, on 2026-10-27 at `8cf0c88`, failed one isolation item, which TICK-271 fixed. At `8ef35f2`, operator onboarding answered `401` because `nava-onboard` still trusted the pre-TICK-160 issuer; TICK-278 fixed that.

**Status: READY (2026-10-28).**

## What you can now do

- An operator creates a new tenant with one API call. It creates the tenant's sign-in organization, connects the tenant's own identity provider, and invites the tenant's first admin.
- Sending the same call again with the same key is safe. You get the same answer and nothing is created twice.
- The invited admin signs in with their company account and sees only their own tenant.
- Someone who already signs in through another tenant's identity provider cannot be made this tenant's admin.

## Try it (about 10 minutes)

You need Docker and Go 1.26. Start from a clean checkout of `TICK-46/operator-onboards-tenant`. The stack has no public identity provider, so the walkthrough is one recorded run of the harness. It stands in a Keycloak test realm for each tenant's company IdP.

1. Start a fresh stack (about 5 minutes the first time, while the images build):
   ```sh
   make dev-ca
   export COMPOSE_PROJECT_NAME=s4-try NAVA_CORE_PORT=28080 KEYCLOAK_PORT=28081
   docker compose down -v
   docker compose build && docker compose up --wait
   ```
2. Run the acceptance walkthrough (about 70 seconds):
   ```sh
   NAVA_S4_EVIDENCE=$PWD/s4-evidence.md \
     go test -tags 'identitystack acceptance' -count=1 -v -run '^TestS4Acceptance$' ./internal/identitystack
   ```
   It signs in as the seeded operator `operator@nava.test`. It onboards tenants A, B and C, each with its own domain `onb-<8 hex>.test` and its own IdP realm. It signs both admins in through their tenant's IdP and calls `GET /api/v1/me` with each admin's token.
3. Open `s4-evidence.md`. Every request and response is there, under one heading per criterion, with tokens left out and the IdP client secret redacted. Every line reads **PASS**.
4. To call onboarding by hand, use the operator command in `docs/operator/onboarding.md` ("Configuration (nava-onboard)"). It passes the token through the environment, never on the command line.
5. Clean up with `docker compose down -v`.

## Acceptance criteria

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | The call returns 201 with the tenant id; repeating it with the same key returns the same body and creates nothing new | ✅ | [evidence](evidence/operator-onboards-tenant.md), "Criterion 1", for tenants A and B |
| 2 | The new admin signs in and is scoped to the new tenant | ✅ | [evidence](evidence/operator-onboards-tenant.md), "Criterion 2" |
| 3 | Inviting a user already federated to another tenant's IdP is rejected | ✅ | [evidence](evidence/operator-onboards-tenant.md), "Criterion 3" |

**Criterion 1.** For each of tenants A and B, the first call is `201` with `{"state":"active","tenant_id":"<id>"}`, and the id is the one derived from the operator and the key. A replay with the same key and body is `200` with a byte-identical body. The counts of tenants, bindings, email domains, invites and Keycloak organizations are the same before and after the replay. The same key with a different body is `409 idempotency_key_reuse` and creates nothing.

**Criterion 2.** Each admin signs in through their tenant's IdP. The token's `organization` claim names exactly that tenant's organization. After one `nava-groupsync` cycle, `GET /api/v1/me` is `200` with that tenant's id and IdP alias, the admin is in `/t/<tenant>/`, and the invite is consumed.

**Criterion 3.** Tenant B's IdP signs in a user whose address is in tenant C's domain. That user now exists, linked to B's IdP only. The operator then onboards tenant C with that user as admin. The results:
- No invite is written for them in C.
- Their sign-in through C's IdP is refused. Keycloak says: "User with email ... already exists."
- After a sync cycle they are not in C's group or organization and hold no link to C's IdP.
- Their token never resolves tenant C. `/me` answers `401` even when the request names C.

As a control, another user of C's IdP signs in, which shows the refusal is about the account and not a broken IdP.

## Isolation check

| Tried | Result |
|---|---|
| B's admin calls `/me` naming A, with `X-Tenant-Id` and with `?tenant_id=`, and A's admin does the same naming B | ✅ Always the caller's own tenant |
| Each admin's membership in the other tenant | ✅ Not in the other's group or organization, and no link to the other's IdP |
| B's admin replays A's onboarding request on `nava-core` (the tenant ingress) | ✅ `501`; nothing changes for A |
| A new tenant claims A's email domain (TICK-271) | ✅ `409 domain_claimed`, and the body names neither tenant A, its organization nor its IdP. A keeps the domain. The refused tenant has no `tenants` row and no Keycloak organization, and no second invite is written |

## Not included

- **The invitation email.** The story's "How to try it" step 2 says to accept an invitation mailed to the dev mail catcher. The stack has no mail catcher and sends no email. Since TICK-252, the invite is a pending row, and the admin signs in through their company IdP with no password to set. The story text predates that change; the CTO confirmed on 2026-10-27 that TICK-252 supersedes it.
- **Abandoning a half-onboarded tenant.** There is no operation for it (AR-036).
- **An operator UI.** Onboarding is API only.

## Known limits

- **A refused invite is answered like a sent one.** In criterion 3, onboarding C returns `201`, and C is `active` with no admin who can sign in. The operator is not told. This is the documented design (`docs/operator/onboarding.md` step 5). If you want the operator to see it, that is a new requirement.
- **A replay answers `200`, not `201`**, with the same body. This is what the operator reference documents.
- **The new admin's `/me` works only after the next group sync cycle**, at most 5 minutes after their first sign-in. The walkthrough runs one cycle by hand.
- **The tenant's IdP URLs must be public https.** On the stack, the harness points them at a test realm afterwards, as the Keycloak master admin.
- **Squatting (AR-037).** An IdP that is not limited to its own domains can claim an address in another tenant's domain first. That tenant's admin is then refused, which is criterion 3 seen from the other side.
- **The onboarding stack tests have no CI step of their own yet** (TODO-012). `make stack-test-identity` runs them, so CI covers them today.
- **One intermittent failure.** In a local unit run of `internal/identity/keycloak`, the full suite failed once under load and then passed 5 times in a row. CI is green.
