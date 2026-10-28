# Review: T26 deprovisioning and identity-boundary tests (TICK-82, PR #153)

Reviewer: Quality, Testing & CI Advisor (first reviewer; Principal Software Engineer is second, D-P10).
Head reviewed: `53d5bb8`, base `TICK-47/group-changes-within-10-minutes`.
Read against: the TICK-82 contract, PLAN-001 D-P9 / D-P10, ADR-001 A.1, and the S4 branch `TICK-46/operator-onboards-tenant` (`internal/server/operator.go`, `api/openapi.yaml`).

**VERDICT: APPROVE.** No must-change. Four should-change items and one taste item are listed below. None blocks the merge.

## The four questions

### 1. Case 6: is the guard strong enough? Yes.

- `operatorRoutesInContract()` (`internal/tenancy/isolation/boundary_paths.go:530`) looks for `/api/v1/operator/` in the embedded `api/openapi.yaml`. If the file can't be read, it returns true, so it fails closed.
- On S4 that string is present (`/api/v1/operator/tenants`, `api/openapi.yaml:31`). When S4 is merged into S5, T26-6 fails in the `integration` job with a message that names the fix. It can't pass silently on a 404.
- On this branch the test proves both inputs to S4's 403 branch (`operatorAuth`, `internal/server/operator.go` on S4): `ResolveOperator` returns `ErrInvalidToken` for the tenant admin, and the tenant resolver accepts the same token (`GET /api/v1/me` returns 200 for tenant A). It also has a positive control: the allow-listed operator is accepted. So it covers the essence of the contract before the route exists.
- TODO-015's fix ("wire `server.WithOperator`, require 403") matches what S4 really does: a valid tenant token on an operator route gets 403, and an invalid token gets 401.

### 2. Case 8: does asserting both 401 variants still pin the contract? Yes.

- The test has two rows, each asserting an exact code: the real-shape token (audience `nava-operator-api`) gets `invalid_token`, and a mis-mapped token (also audience `nava-api`) gets `tenant_unresolved`. Each row checks `"<code>"` in quotes, so neither can satisfy the other. Each row has the operator-accepted control and a check that no tenant id leaks.
- The minted token matches reality. `internal/identitystack/stack_test.go:322` asserts that a real operator token's audience includes `nava-operator-api` and not `nava-api`.
- The contract text itself is off. See finding 3.

### 3. Stack-only cases 3, 4, 5 and 7: does the ci-unit guard work? Yes, verified.

- `TestStackPathsExist` has no build tag, so it runs in `make test` (the `unit` job). It parses `internal/identitystack/*_test.go` with `go/parser`, which ignores build tags.
- Mutation check: I renamed `TestTokenExchangeStaysDisabled` to `...X`, and `go test -run TestStackPathsExist ./internal/tenancy/isolation/` failed with `path "T26-5 token exchange stays disabled": TestTokenExchangeStaysDisabled is not a test in internal/identitystack`. I reverted the rename. It also fails when no Stack path is registered.
- `make stack-test-identity` runs the whole package with no `-run` filter. None of the four tests calls `t.Skip`. Remaining gap: finding 4.

### 4. GitHub Actions: all green.

`gh pr checks 153` shows all 8 jobs passing: changes, lint, unit, integration, temporal-server, stack, stack-db and stack-internal. The `integration` log shows T26-1…8 running (cases 3, 4, 5 and 7 as STACK). The `stack` job's "Realm nava, operator custody and the Keycloak adapter" step reports `ok internal/identitystack`.

## Findings (most severe first)

| # | Severity | Blocks? | Where | Finding | Fix |
|---|---|---|---|---|---|
| 1 | Breaks under a config change | should-change | `boundary_paths.go:199` (`withinBound`) | D-P9's sync half is checked against `groupsync.DefaultInterval`, the library default. The deployed bound is `config.MaxGroupsyncInterval`, enforced at `internal/platform/config/groupsync.go:78`. If the config cap is raised to 10 min while the library default stays at 5, the 10-minute bound is broken and the test still passes. | Check `config.MaxGroupsyncInterval` too, or instead. |
| 2 | Missing test | should-change | `boundary_paths.go:472` (case 6) | The tenant admin is refused only because of its audience (`nava-api` alone). Nothing covers a tenant-admin token that also carries `nava-operator-api`, which is the same mapper-mistake scenario case 8 already covers in the other direction. That token would rely on the operator allow-list. | Add a row: tenant admin with audiences `nava-api` and `nava-operator-api` plus an org → `ResolveOperator` is still refused. |
| 3 | Spec drift (spec is wrong) | no, route to contract owner | TICK-82 contract, "Operator boundary" | The contract says an operator token on a tenant endpoint gets 401 `tenant_unresolved`. With the realm as built (D-P12 audience split), a real operator token gets 401 `invalid_token`. `tenant_unresolved` only appears for a mis-mapped token. The code is right. The contract text should say "401 (`invalid_token`; `tenant_unresolved` if the token also names `nava-api`)". | The Technical Lead amends the TICK-82 contract text. No code change. |
| 4 | Missing test (narrow) | should-change | `registry_isolation_test.go:91` (`TestStackPathsExist`) | The guard finds a test by name in any `_test.go` file, whatever its build tag. A test moved to a file without `//go:build identitystack` would still pass the guard but stop running in `make stack-test-identity`. | Also require the declaring file's build constraint to include `identitystack` (read `af.Comments` / `//go:build`). |
| 5 | Preference | taste | `Makefile` `stack-test-identity` | The run is not verbose, so the CI log shows only `ok internal/identitystack`, not the four T26 test names. Evidence per case means re-running with `-v`. | Optional: `-v`, or `-json` through a reporter like `ci6report`. |

## Merge prerequisites (not review findings)

- The CTO's sign-off on the isolation-registry change (D-P10) has to be recorded before merge.
- TODO-015 is recorded in `docs/delivery/todo.md` with its code comment. When S4 is merged into S5, T26-6 fails until it is switched to 403.
- The accepted at-scale risk the Security Reviewer raised (unverified email change through the account API once client scopes return) is written up in `docs/security/deprovisioning-identity-boundary.md`. Per the repo rule, an accepted risk also needs an `AR-NNN` row in `docs/security/accepted-risks.md`. That routes to the Security & Multi-Tenancy Reviewer; it is not part of this review.
