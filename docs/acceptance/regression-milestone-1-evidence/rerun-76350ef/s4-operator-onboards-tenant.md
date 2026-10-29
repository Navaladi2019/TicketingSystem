# S4 regression re-run (TICK-46 kit), main @ 76350ef, 2026-10-28, stack nava-regr2

Proves fix #272 (TICK-319: `onboardPostRaw` rename, the `acceptance` tag compiles).

Kit step 2 verbatim (no overlay this time):
```
NAVA_S4_EVIDENCE=$PWD/s4-evidence.md \
  go test -tags 'identitystack acceptance' -count=1 -v -run '^TestS4Acceptance$' ./internal/identitystack
--- PASS: TestS4Acceptance (60.24s)
ok  github.com/Navaladi2019/TicketingSystem/internal/identitystack 73.167s
```
Harness output: `s4-harness-evidence.md`: 43 x **PASS**, 0 FAIL, no tokens (`grep eyJ` = 0), 8x `"clientSecret": "<redacted>"`.

| # | Criterion | Result | Evidence |
|---|---|---|---|
| 1 | 201 + tenant id; same-key replay same body, nothing new | PASS | A `0d216d6e-…`, B `de775309-…`: `201 active`; replay `200` byte-identical; counts equal before/after; same key other body `409 idempotency_key_reuse` |
| 2 | new admin signs in, scoped to new tenant | PASS | brokered sign-in per tenant IdP, `organization` exactly `t-<tenant>`; `/me` `200` own tenant + `idp-<tenant>`; invite consumed |
| 3 | user federated to another tenant's IdP cannot be invited | PASS | no invite in C; Keycloak "User with email lead-…@onb-d8bdba37.test already exists"; not in C's group/org; `/me` `401`; C's control admin signs in |
| iso | B cannot see/change A's onboarding | PASS | `/me` naming the other tenant (header, query) -> own tenant; no cross membership/links; replay on nava-core `501`; A's domain claimed by a new tenant -> `409 domain_claimed`, no rows, no org |

Step 4 (operator call by hand): exercised in `cross-story.md` 4a (`201`, replay `200`).

Kit drift (Git Bash on Windows): `NAVA_S4_EVIDENCE=$PWD/...` expands to `/c/Users/...`, which Go on Windows reads as `C:\c\Users\...`; the evidence file landed in `C:\c\Users\...\TicketingSystem\s4-evidence.md`, not in the repo. Moved by hand. Suggest `$(pwd -W)` or a relative path in the kit.
