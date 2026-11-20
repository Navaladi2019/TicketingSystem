# Review: TICK-996 / PR #778 — format-1 rule over the child cap is 422, not 500

Reviewer: Quality, Testing & CI Advisor · Base: `TICK-935/composable-access-rules` · Head: `1ff588df`

Spec: finding F3 in `docs/security/s30-composable-access-rules.md`; task TICK-996.

## Verdict

APPROVE (code). No Bug or Breaks-under findings.

## What was checked

- `prepareFormat1` now runs `Decode(enc)` on the exact bytes `publishRaw` later decodes. So `Prepare` can no longer accept a format-1 document that `Decode` refuses, which was the F3 500. It covers every Decode refusal, not just the child cap.
- `PublishIfNone` (onboarding) goes through `Prepare`, so the same check turns its plain error into an `InvalidPolicyError`.
- Pointer mapping reuses `at`, so `/policies/<i>/...` becomes `/rules/<i>`, the same as the cap problems from `finish`. The PR uses `/rules/<i>`, not F3's suggested `/rules/<i>/where`. That is consistent with how the other caps are reported, and it is fine.
- Non-`DecodeError` errors fall back to `/rules type`, which is a 422, not a 500.
- Unit test `TestPrepareFormat1OverTheChildCapIsChildCount` fails before the fix (red-first, per the PR). The integration test asserts 422, `/rules/0 child_count`, and that nothing was written (`acl_versions`, audit).

## Findings

1. **Missing test** (low): no test asserts that `PublishIfNone` with an over-cap format-1 rule returns `*InvalidPolicyError`. F3 flagged this path as "read from the code, not tested". A unit or integration test calling `PublishIfNone` with 16 `where` and checking `errors.As(err, &InvalidPolicyError{})` would close it. This is optional and does not block the merge.
2. **Preference**: Decode's problems are returned without `finish`'s sort/compact, and before the byte-size check. With one rule this makes no difference. With several bad rules, the problem order is Decode's order. This is taste and needs no change.

## CI

GitHub `ci` run 44634136748:
- Attempt 1: `TestS30VoidedPolicyGrantsNothingAnywhere` and every `internal/server` and `internal/authz` test passed. The run went red only on the timing budget in `TestI5Format2Agreement` (66.5s against a 60s budget, `internal/tenancy/isolation`). That test does not touch format-1 Prepare. `coverage` failed only because the integration artifact was missing.
- Attempt 2 (failed jobs re-run): `TestI5Format2Agreement` passed in 48.8s and the S30 test passed. The integration job was then cancelled at its 15-minute hosted-runner `timeout-minutes`.

Both failures are CI capacity problems, not this change. They are tracked separately.
