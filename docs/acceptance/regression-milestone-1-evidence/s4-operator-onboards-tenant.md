# S4 regression evidence (TICK-46 kit), main @ 4392c26, 2026-10-28, stack nava-regr

## FAIL as written: the kit's harness does not compile on main
Kit step 2, verbatim (ports 28080/28081):
```
NAVA_S4_EVIDENCE=$PWD/../s4-evidence.md go test -tags 'identitystack acceptance' -count=1 -v -run '^TestS4Acceptance$' ./internal/identitystack
internal\identitystack\s4_acceptance_test.go:78:6: onboardPostRaw redeclared in this block
	internal\identitystack\onboard_e2e_test.go:226:6: other declaration of onboardPostRaw
internal\identitystack\s4_acceptance_test.go:251:17: assignment mismatch: 2 variables but onboardPostRaw returns 3 values
FAIL	github.com/Navaladi2019/TicketingSystem/internal/identitystack [build failed]
```
Cause: `1f5a376` (TICK-253, 2026-10-28) added `onboardPostRaw` (3 results) to `onboard_e2e_test.go` (`//go:build identitystack`); `s4_acceptance_test.go` (`//go:build identitystack && acceptance`) already had its own 2-result `onboardPostRaw`. CI never builds the `acceptance` tag, so nothing caught it.

## Workaround run: PASS 43/43
Same command plus `-overlay overlay.json`, which swaps in a copy of `s4_acceptance_test.go` with its helper renamed `s4OnboardPostRaw`. No repo file was changed.
```
--- PASS: TestS4Acceptance (142.15s)   ok ... internal/identitystack 186.971s
s4-harness-evidence.md: 43 x **PASS**, 0 FAIL, no tokens (grep eyJ = 0), clientSecret "<redacted>"
```
Full harness output: `s4-harness-evidence.md`.

| # | Criterion | Result | Excerpt |
|---|---|---|---|
| 1 | 201 + tenant id; same-key replay same body, nothing new | PASS | tenant A `2c47d9fc-…` and B `19985f68-…`: `201 {"state":"active","tenant_id":…}`; replay `200`, byte-identical; counts equal before/after (`4 tenants, 6 bindings, 2 email domains, 2 invites, 1 Keycloak organizations`); same key, other body `409 idempotency_key_reuse` |
| 2 | new admin signs in, scoped to new tenant | PASS | brokered sign-in per tenant IdP, `organization` claim exactly `t-<tenant>`; after one sync `/me` `200` with own tenant and `idp_alias idp-<tenant>`; invite consumed |
| 3 | user federated to another tenant's IdP cannot be invited | PASS | no invite in C; Keycloak "User with email lead-…@onb-b7a11e16.test already exists"; not in C's group/org, no link; `/me` `401`; control user of C signs in |
| iso | | PASS | B's admin naming A (header, query) -> own tenant; no cross membership/links; replay on nava-core `501`; A's domain claimed by a new tenant -> `409 domain_claimed`, no rows, no org |

Step 4 (operator call by hand with the `docs/operator/onboarding.md` wget form): exercised in `cross-story.md` 3a: `201`, replay `200`.

Rough edges: the harness deletes the onboarded Keycloak users/orgs/IdPs at cleanup but leaves their `tenants`/binding/domain rows, so the live sync then tombstones the two admins (`principals_tombstoned:1`). That is fine for a throwaway run but means its tenants cannot be reused for later checks. The criterion 3 page excerpt starts mid-JavaScript ("efox = true; nava We are sorry...").
