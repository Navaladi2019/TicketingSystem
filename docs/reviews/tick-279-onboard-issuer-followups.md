# Review: T21j PR #217 review follow-ups (TICK-279, PR #218)

Reviewer: Quality, Testing & CI Advisor. Head `c5db439`, base `TICK-46/operator-onboards-tenant`.

**Verdict: APPROVE** at `c5db439`. Finding 1 (the blocking test bug) is fixed and all 8 CI jobs are green (run 40592131964). The first round, at `09aa7bb`, was CHANGES REQUESTED.

## Acceptance criteria

| Criterion | Result | Evidence |
|---|---|---|
| Fetch base URL rejects a path and a query, with unit tests for both services | Pass | `onboard_test.go` (nava-onboard) and `TestLoadFetchBaseURL` in `config_test.go` (nava-core, added at `90420ed`) cover path, query, path+query and the trailing slash. |
| Forged-issuer onboarding test passes in the `stack` job | Pass | `TestForgedIssuerTokenRejectedByOnboard` passed in `stack` (run 40592131964, `c5db439`). It failed at `09aa7bb` (finding 1). |
| `task test`, `task lint`, GitHub Actions green | Pass | Run 40592131964: all 8 jobs green |
| Compose test tidy | Pass | One `composeEnv` read per service, no one-element loop |

## Findings

1. **Bug (blocking), fixed at `c5db439`:** the test now mints the control and forged tokens as `operator@nava.test` through `nava-operator-cli` (`forgedOperatorToken`). The control checks for the pinned `iss` and requires neither 401 nor 403. Original finding: the control token can never pass nava-onboard, so the test fails on every run.** The control is alice's `nava-web` password-grant token (`forgedToken`). `operatorAuth` (`internal/server/operator.go:100`) first calls `ResolveOperator`. That requires audience `nava-operator-api`, realm role `nava-operator` and an allow-listed (iss, sub), so alice's token fails with `ErrInvalidToken`. The handler returns 403 only if `tenants.Resolve` succeeds for the same token. Otherwise it returns 401 `invalid_token`. On this stack alice's token is not resolved there, so the control gets 401. The test comment's claim ("alice is no operator, so it is refused later with 403") does not match the handler. Even where it holds, a 401-vs-403 split would depend on the tenant binding, not on the issuer.
   **Fix:** mint the control and the forged tokens as the operator. Use `client_id=nava-operator-cli`, `operator@nava.test` / `operatorPass`, against the same `/realms/nava/.../token` endpoint with the forged Host and headers. That client has `directAccessGrantsEnabled: true` in `deploy/compose/keycloak/import/realm-nava.json`. A parameterised `forgedToken(t, host, header, form)`, or a small `forgedOperatorToken`, keeps the API test unchanged. Then the only difference between the control and a forgery is `iss`. The control must get past auth: assert that it is not 401 or 403 (it onboards a fresh tenant from `newOnboarding`, the same as the e2e tests). Each foreign-`iss` operator token must get 401. Fix the doc comment on the test to match.
2. **Missing test, closed at `90420ed`.** No nava-core unit test for the fetch base URL. `TestLoadFetchBaseURL` now covers it through `config.Load`.
3. **Breaks under a Keycloak served under a path prefix (non-blocking, record only; kept by design and documented in the PR body).** `baseURL` also reads `NAVA_KEYCLOAK_URL` (`onboard.go:92`, `groupsync.go:53`) and `NAVA_KEYCLOAK_MANAGEMENT_URL` (`config.go:139`), so those now reject a path as well. Every current value is host-only (compose and `.env.example`), so nothing breaks today. A deployment that puts Keycloak behind `KC_HTTP_RELATIVE_PATH=/auth` would fail at startup. That is acceptable, because it fails closed and loudly. If it is intended, mention it in `docs/architecture/nava-core-skeleton.md`; if not, apply `noPath` only to the fetch base URL.
4. **Preference, addressed at `c5db439`:** a `t.Logf` now reports issued and foreign-`iss` counts. Original note: the foreign-`iss` branch is unlikely to run. With `KC_HOSTNAME` pinned (TICK-160), Keycloak issues every forged request under the pinned issuer, so every subtest takes the `iss == pinned` early return. That is the same in `TestForgedIssuerTokenRejectedByAPI`. The test then proves the pin, not that nava-onboard rejects a foreign `iss`. A `t.Logf` with the count of foreign-`iss` tokens actually sent would make that visible in the log. The unit-level issuer check in the keycloak adapter's contract tests remains the real guard.

`noPath` and the `isBaseURL` split are correct. Issuer URLs keep `/realms/nava`, and the trailing `/` that `noPath` allows is trimmed by the callers' `TrimRight`. Tenant isolation was not reviewed here (Security & Multi-Tenancy Reviewer).

## CI

- Run 40589429658 (pull_request, head `09aa7bb`): `stack` failed, all other jobs green. Log: `--- FAIL: TestForgedIssuerTokenRejectedByOnboard`, `issuer_pin_adversarial_test.go:222: control token refused with 401`.
- Run 40592131964 (pull_request, head `c5db439`): all 8 jobs green (changes, lint, unit, integration, temporal-server, stack, stack-db, stack-internal). `issuer_pin_adversarial_test.go` ran 3 of 3 tests.
