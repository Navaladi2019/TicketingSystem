# Review: TICK-475, CI-8 mutation control accepts only the oracle check (PR #387)

Reviewer: Quality, Testing & CI Advisor. Date: 2026-11-02.
Scope: `internal/tenancy/isolation/records_ci8_paths.go` (commits `256aa9c` and `fcc4181`), against the TICK-475 acceptance criteria and the TICK-87 review (`docs/reviews/tick-87-ci8-record-routes.md`).

## Verdict

**Approve.** There are no Bug, Breaks-under or Spec-drift findings. All three acceptance criteria are met. One Missing test finding (low) and one Preference are listed below. Neither one blocks the merge.

## Acceptance criteria

| Criterion | Result |
|---|---|
| `ci8Mutation` accepts only the oracle failure, matched with `errors.Is` on a sentinel. A `control:` error fails the mutation control. | Met. `ci8MutationThrough` returns nil only when `errors.Is(err, errCI8Oracle)`. A nil error or any other error fails the mutation control. |
| Red, then green: a `ci8Leaky` variant that also intercepts the random UUID makes the mutation control fail. | Met. In (g), `ci8LeakyAll` gives the random UUID 412 or 428, so (e) and (f) fail on their `control:`. Under the old `ci8Mutation`, which took any error as a catch, (g) fails. The author reported this red at `256aa9c`, and the green after the fix. |
| Add-only change (D-P10). | Met. (a) to (f) are unchanged in intent, (g) is added, and (d) is still pending. Nothing was removed or skipped. |

## What I checked

- **Where the sentinel is wrapped.** `errCI8Oracle` is wrapped only where tenant A's response is compared with the random UUID's: the code mismatch, the body mismatch, the header-set mismatch and the header-value mismatch in `ci8Same`, plus the 412 check in `ci8ETagCases`. It is never on a `control:` error, a DB error from `ci8Unchanged`, or the checks that look only at the random UUID's response (`instance`, `ETag` on 404). Those checks are not an oracle, so leaving them unwrapped is correct.
- **Control order in `ci8ETagCases`.** Before this change, a 412 for tenant A's record would short-circuit before `ci8Same` checked the random UUID's 404. That is how a fully-intercepting handler could pass as a catch. The new early `missing.code != 404` control closes that gap. `ci8NoIfMatchCases` goes through `ci8Same`, whose first case is already the control, so it needed no change.
- **`ci8LeakyHandler(all=true)`.** It answers every PATCH with 428 when If-Match is missing and 412 otherwise, including when the version matches. So the random UUID can never return 404 through it, and (g) is deterministic. Like `ci8Leaky`, it answers before the real handler, so it writes nothing, and `ci8Unchanged` at the end of (g) still holds.
- **The positive control in (g).** `ci8Mutation` through `ci8Leaky` must pass first. That ties a (g) failure to `ci8LeakyAll`, not to a broken case.
- I ran `go vet` and `go test ./internal/tenancy/isolation/` locally, and both are clean. The DB-backed paths run only in the CI `integration` job, which I did not run locally (CI execution rule).

## Findings

### 1. Missing test (low): (g) does not check *why* the mutation control failed

`ci8MutationRejectsControl` only asserts that `ci8MutationThrough(..., ci8LeakyAll(e), ...)` is non-nil. That function has two failure branches. One is the case passing against the leaky handler (`err == nil`). The other is a non-oracle failure (`!errors.Is(err, errCI8Oracle)`). (g) is meant to pin the second branch. Suppose a later edit to `ci8LeakyHandler` let (e) or (f) pass through `ci8LeakyAll`: (g) would stay green through the first branch and no longer prove the rejection of the `control:` error. Today that cannot happen, because `ci8LeakyAll` never returns 404 for the random UUID. So this finding is about future regressions only.

Suggested fix (test code, optional): add a second sentinel `errCI8NotOracle`, wrap it in the `!errors.Is` branch of `ci8MutationThrough`, and have (g) require `errors.Is(err, errCI8NotOracle)`. A string match on "before its oracle check" would also work, but it is weaker.

### 2. Preference: (g) re-runs the `ci8Leaky` mutation of (e) and (f)

The positive control in (g) repeats work that (e) and (f) already do, which adds about four PATCH round trips. It costs little and makes (g) self-contained, so keeping it is fine.

## Routing

No isolation concern: this PR changes only the test harness, and no product code changed. The Postgres & Data Modeling Specialist's D-P10 review is still required before merge.
