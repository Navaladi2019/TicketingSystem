# Review: TICK-304, fail tests on a keycloak.New call outside tokenValidators() (PR #258)

Reviewer: Quality, Testing & CI Advisor. Change: `deploy/compose/validators_test.go` (new, test only).

**VERDICT: APPROVE.** No `must-change` findings.

## Against the acceptance criteria

| Criterion | Result | Evidence |
|---|---|---|
| Every non-test `keycloak.New` call site is found from the source (AST), and the test fails if one is outside `tokenValidators()` | Met | `loadModule` + `verifierRefs` parse every non-test `.go` file with `go/parser`, ignoring build tags. `checkVerifierBinaries` walks the in-module import graph from each `main` package. |
| A deliberately unlisted call site makes the test fail | Met | The PR description shows `cmd/nava-rogue`. I ran my own mutation (below). |

## What I checked

- **Detection forms:** a direct call, a renamed import, a function value, a dot import, and an unqualified `New` inside `keycloak` are all detected. Selector fields, method and func declarations, struct fields, and `errors.New` are all ignored. `TestVerifierRefs` covers each case.
- **Transitive reach:** a call inside a library counts against every binary that imports it, and so does a wrapper inside `keycloak` itself, because that package's own `Verifiers` propagate to its importers. `TestCheckVerifierBinariesCatchesUnlisted` covers direct, through-a-library and stale-entry cases.
- **No false positive today:** `cmd/nava-groupsync` imports `keycloak` but only uses `NewDirectory`, so it is correctly not flagged. `keycloaktest` is imported only from `_test.go` files, so it reaches no binary.
- **Stale list:** a name in `tokenValidators()` with no matching binary fails the test.
- **Package moved:** if `keycloakPkg` no longer exists, the test fails loudly instead of passing with nothing to check.
- **Runs in `task test`:** the CI `unit` job (`go test -race`) ran `deploy/compose` and it passed.

## Local verification

- `go test -race -count=1 -run 'Verifier|TokenValidator' ./deploy/compose/`: pass.
- Mutation: I added `cmd/nava-groupsync/zz_mut.go` with `import kc ".../keycloak"; var _ = kc.New` (a renamed import used as a function value, in an existing unlisted binary). The build succeeded and the test failed with `binary nava-groupsync ... builds a token verifier at cmd/nava-groupsync/zz_mut.go:5 but is not in tokenValidators()`. I removed the file afterwards.

## Findings

None at `must-change`.

- **taste:** with a dot import, any identifier spelled `New` (a local variable, or a key in a keyed composite literal) is counted as a reference. This can only produce a false positive, never a false negative, and there are no dot imports of `keycloak` today. Leave it as is.
- **taste:** the `go.mod` module line is parsed by hand. That is fine for this module, and it avoids adding `golang.org/x/mod` as a dependency.

## Deliberately not covered

- Reaching `keycloak.New` through reflection or code generation at build time. Neither happens in this codebase, and the cost of catching them is out of proportion to the risk, which the issue rates as low.
