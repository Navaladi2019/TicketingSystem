# Review: T19 CI-5 adversarial identity suite (TICK-75, PR #159)

Reviewer: Quality, Testing & CI Advisor. Reviewed at `752b7a4`, against ADR-001 Annex B CI-5 and the TICK-75 contract.

**VERDICT: APPROVE.** Nothing blocks (no `must-change`). Three non-blocking items are listed below.

## What was checked

| Check (from the Technical Lead) | Result | Evidence |
|---|---|---|
| Each of the 14 CI-5 cases is one named test | Yes | `contracttest.CI5Cases()` lists 14 ids (CI-5.01 to CI-5.14), in Annex B order. `TestCI5` (`internal/identitystack/ci5_isolation_test.go`) tags exactly one case per id: 5 resolver cases, 1 directory case, 8 scenarios. `Suite.Run` names each one `[CI-5.NN] …`. |
| `ci5-report.md` lists them all | Yes | `tools/ci/ci5report` loops over `CI5Cases()`, not over the tests that ran. A missing case is printed as `MISSING` and fails the run. A skipped case, a failed one, or one that did not finish also fails the run, and so do an untagged skip and a build error. All of these are covered in `tools/ci/ci5report/main_test.go`. |
| The "every listed case has a test" guard fails when a case is missing | Yes, at two layers | (1) `Suite.Require: "CI-5"` adds a `coverage` subtest that calls `Errorf` for each entry it does not cover. I checked this by mutation: a temporary test ran `Suite.Run` with CI-5.06 left out, and it failed with `no case covers CI-5.06 …` (the test was removed afterwards). (2) `ci5report` reports a case with no tagged subtest as `MISSING` (`TestMissingCaseFails`). A tag counts only on the last name element (`TestTagOnlyOnLastElement`), so a parent test named after a case proves nothing. |
| The 26.1.2 `kc.org` fixture is skipped by default | Yes, twice | `TestKC2612FixtureOrgPickerTamperSucceeds` calls `t.Skip` unless `NAVA_CI5_KC_FIXTURE_URL` is set. `make stack-test-identity` also leaves it out with `-skip '^(TestCI5\|TestKC2612Fixture.*)$'`. Only `make identity-attacks-fixture` sets the URL. The fixture checks `serverinfo` for version 26.1.2, then fails unless the tamper gets a token for the organization the user does not belong to. So a passing fixture means the CI-5.06 attack works on 26.1.2. |

Other checks:

- **A case cannot pass by not running.** CI-5.06 calls `Fatal` if Keycloak shows no organization picker. CI-5.04 has a control: the realm's own key must verify first, so a broken signature cannot make the case pass. `TestCI5` sets up the no-organization, two-organization and brokered tokens, and calls `Fatal` if any of them is the wrong shape. The sync calls `Fatal` if the group sync was skipped or returned an error.
- **The isolation registry edit** (`registry.go`, `ci5_paths.go`, `http_isolation_test.go`) is narrow. A `Token` shape passes only on 401, and every other path keeps its baseline. The GitHub `integration` job passed with these paths in it. This still needs CTO sign-off under D-P10. The isolation-boundary review is left to the Security & Multi-Tenancy Reviewer.
- **Pins:** the Keycloak row records the negative fixture, and `check-pins` passes. The fixture script reads the tag from `docs/pins.md`, so the test code does not hard-code an image tag.
- **Run locally:** `go test` on `./internal/identity/...`, `./tools/ci/ci5report`, `./tools/ci/checkpins` and `./tools/ci/cichanged` passed. `go vet -tags identitystack ./internal/identitystack` passed. `go run ./tools/ci/checkpins` passed. I did not run `make identity-attacks` (that is a stack run).

## Findings (most severe first)

1. **should-change: the CI-5.12 cleanup deletes the wrong tenant's row.** `internal/tenancy/isolation/ci5_paths.go:90`. If a duplicate insert ever succeeds, the cleanup deletes the row `tenant_id = TenantB` whatever `tenant` the loop is on. When the loop is on TenantA, the stray TenantA row stays in the suite's database for the paths after it. The case still fails correctly, so nothing is wrong today. Fix: pass `tenant` instead of `TenantB`.
2. **should-change: no test checks that `Suite.Run` fails a suite with a missing case.** `internal/identity/contracttest/contracttest_test.go:10`. `TestUncovered` tests `Uncovered()` but not that `Run` calls it, so a refactor could drop the `coverage` subtest without any unit test failing. `ci5report` would still catch it in `make identity-attacks`, but not in a future adapter's own suite. Fix: run `Suite.Run` from a child test (`go test -run` in a subprocess, or `testing.RunTests`), with one CI-5 case missing, and assert that it fails.
3. **taste: `docs/pins.md` changes from CRLF to LF line endings in this PR.** The diff is 302 lines for one changed row. That is harmless, but reviewers have to use `--ignore-cr-at-eol` to see the real change.

## Not for this review (already tracked)

- **TODO-016:** CI-5 does not run in GitHub Actions yet, and T19 cannot close until it does. The proposed step text in `docs/delivery/todo.md` is correct: it runs `make identity-attacks` after `stack-test-identity`, then uploads `ci5-report.md` with `if: always()` and `if-no-files-found: error`.
- **CI on #159:** `changes`, `lint`, `unit` and `integration` pass. `stack`, `stack-db`, `stack-internal` and `temporal-server` were still pending when I checked.
