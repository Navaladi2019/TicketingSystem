# Review: TICK-348 allow `nava.interpreter` on `nava-tq` (PR #305)

**Verdict: APPROVE.** Nothing blocks the merge. One finding is a small follow-up for the test helpers. The other is taste.

Reviewed head `1dfaf98`. Read first: the TICK-348 contract (PLAN-003 D-3-11, ADR-001 A1 RS5), `temporal-server/authz/policy.go`, `authz/authorizer.go` (`checkWorkflowType`, `checkNormalTaskQueue`), and the existing CI-6 helpers in `authz/commands_test.go` and `authz/workload_test.go`.

## What was checked

- **The policy change is one entry.** `platformWorkflowTypes` gains `nava.interpreter`. `platformTaskQueues` is still exactly `{nava-tq}`, and `TestInterpreterTypeIsAllowListed` pins that, so a second queue cannot slip in. No verb table changed, so no identity can call an API it could not call before.
- **Exact match.** `checkWorkflowType` uses `inSet`, so the look-alikes (`nava.interpreter2`, `.x`, a different case, leading or trailing space) are denied on start, signal-with-start, child and continue-as-new. Each case asserts `ReasonWorkflowType`, not just "deny", so a deny for some other reason cannot pass the test by accident.
- **Verbs stay with their identities.** The worker cannot start, signal or signal-with-start. The relay cannot poll workflow or activity tasks and cannot complete them. Core cannot start, signal or poll. Each is asserted as a gate-level `ReasonAPI` deny with exactly one recording.
- **E1 is unchanged for the new type.** These are all still denied with the right reason: a child in tenant B, a child with no prefix, continue-as-new with no header, continue-as-new with tenant B's header, and a relay start with no tenant prefix.
- **The positive tests really test the new entry.** Start, signal-with-start, child and continue-as-new all name `nava.interpreter`, and they fail with the line reverted. The `SignalWorkflowExecution` positive case names no type, so it passes either way. It is still useful as a regression check (see finding 2).
- **Golden and docs.** `allowlist.golden.md` and `docs/temporal-server.md` both list the new set. The Dockerfile lists no types, so no image file needs to change.
- **Ran locally** (`go test -count=1 ./...` in `temporal-server/`): `authz`, root and `cmd/server` all pass. All 24 `Interpreter` subtests pass, and `go vet ./authz/` is clean. On the first run `cmd/server` `TestServerEndToEnd` panicked inside `temporal.NewServer` on this Windows host. It passed on the rerun. The diff does not touch `cmd/server`, and the `temporal-server` CI job is green on the PR, so I read this as a local flake that has nothing to do with this change.
- **CI:** `unit`, `lint`, `temporal-server`, `integration` and `story-branches` are green on the PR. `stack*` was still running when I checked.

## Findings

### 1. Missing test (minor): the existing CI-6 command tests now quietly run on `nava.interpreter`
`authz/commands_test.go:51`: `platformType()` returns `sortedKeys(platformWorkflowTypes)[0]`. That used to be `nava.smoke.Echo`. Now it is `nava.interpreter`, because it sorts first. About 22 existing child and continue-as-new cases now run with the new type. Nobody chose that. The suite still passes and nothing is uncovered today: the smoke types are still checked at `adversarial_sticky_command_test.go:98` and `cmd/server/server_test.go:83`. But which type the CI-6 suite exercises is now decided by alphabetical order. The next type added to the set, or removed from it, will move every one of those tests again, and no diff will show it.
**Fix (test-only, now or in the next policy PR):** pin `platformType()` to a named constant (for example `nava.smoke.Echo`). Or loop the child and continue-as-new allow cases over every entry in `platformWorkflowTypes`. The loop also catches an entry that is added to the set but somehow not admitted.

### 2. Preference: the relay signal positive case does not depend on the new entry
`workload_interpreter_test.go:58`. `SignalWorkflowExecutionRequest` names no workflow type, so this case passes with or without the policy line. That is fine as a regression check. A comment saying so would stop a reader from counting it as evidence for the entry.

## Isolation
I found no isolation concern beyond what the Security & Multi-Tenancy Reviewer already reported: the new type goes through the same D9 and E1 checks as the smoke types.

## How to run

```
cd temporal-server && go test -count=1 ./authz/ -run Interpreter -v
cd temporal-server && go test -count=1 ./...
```

CI: the `ci` workflow on PR #305 (`gh pr checks 305`).
