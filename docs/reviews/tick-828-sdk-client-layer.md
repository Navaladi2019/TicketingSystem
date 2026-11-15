# Review: TICK-828 / PR #665, the SDK's hand-written client layer (P8-23)

- PR: #665 (`TICK-828/sdk-ts-client-layer` → `TICK-801/typescript-sdk`), head reviewed before merge; merge commit `d563166f`.
- Author: Senior Software Engineer. Reviewer: QA.
- Spec: TICK-828 scope and acceptance criteria; ADR-007 D10, D11; PLAN-008 P8-23, contract C6; `api/errors.yaml`.

## Verdict

**APPROVE (merged).** I found no Bug, Breaks-under-load or Spec-drift findings in the code. There are two Missing-test findings. I closed both in this review PR with test code only. There is also one spec gap for the C6 owner and two Preference items. None of them blocks the merge.

## CI

GitHub `ci` run 43581361143 on PR #665: every job passed or was skipped by design (`main-red`, `self-hosted-cleanup`, `temporal-server`).

The green run does not cover the SDK. No CI job runs `sdk:test` yet. That job is TICK-827 (PR #661, CI-E10), which is still open. So I checked the acceptance criteria locally with the package's own scripts: `npm ci`, `npm test`, `npm run lint` in `sdk/ts`.

## What I checked

| Criterion | Evidence |
|---|---|
| Cursor iteration ends | `list follows cursors…` (2 pages) and `list of an empty page ends`. |
| 401: one more `getToken`, one retry, same Idempotency-Key | Test passes. M1 (retry removed): 2 tests fail. M2 (retry rebuilt without body and key): 1 test fails. |
| Second 401 throws | `a second 401 throws and does not retry again`: 2 token calls, 2 requests. |
| 412 → `VersionConflict` | Test passes. M3 (412 mapping removed): 1 test fails. |
| 422 `record_invalid` → `NavaProblem` with `fields` | `fields` comes from the problem's `errors[]` (`pointer`, `rule`). That matches `api/errors.yaml` ("See errors for the failing fields"). |
| 429 → `RateLimited.retryAfter` | Both cases are tested: seconds, and header absent. |
| `Nava-SDK: ts/<version>` on every request | Tested on `get`. See finding 2 for the version source. |
| Size, dependencies | About 190 hand-written lines. No new dependency (`openapi-fetch` was already pinned by TICK-826). |

Local result after this PR: `npm test` 15/15 pass. `npm run lint` is clean.

## Findings

### 1. Missing test: nothing checks that the token is not kept between requests (scope item 1)

The scope says "`getToken()` before each request … never store the token beyond the call". I tried a mutation, M5: cache the token in the client and clear it only on a 401. All 13 original tests still passed. Only the 401 path checked that a fresh token is fetched.

Added test: `each request asks getToken again; the token is not kept between requests`. It runs two list pages and then a `get`, and expects `tok1`, `tok2`, `tok3` with 3 `getToken` calls. M5 now fails 1 test.

### 2. Missing test: `SDK_VERSION` is a hand-copied constant, so the header can go stale

`src/version.ts` hardcodes `1.0.0-alpha.1`, and `package.json` has the same value today. The spec says the header carries the *package* version. A version bump in `package.json` alone would leave `Nava-SDK` reporting the old version, and no test would notice.

Added test: `SDK_VERSION equals the version in package.json`. Changing `version.ts` to `alpha.2` now fails it.

### 3. Spec gap (for the C6 owner, not the code): `update` has no idempotency-key option

The API requires `Idempotency-Key` on PATCH, but C6's `update(type, id, patch, { version })` has no way to pass one. So the SDK generates a fresh UUID on every call, and reuses it only on its own 401 retry. This is safe: if a caller retries after a timeout and the first PATCH did succeed, the new key gets 412 from `If-Match`, so the update is never applied twice. But the caller sees a `VersionConflict` instead of the replayed response. If C6 should offer `{ version, idempotencyKey? }` the same way `create` does, the Solution Architect should amend C6. The code needs no change until then.

### 4. Preference: `list` trusts the server's cursor completely

If the server ever returned the same `next_cursor` again, `list` would loop forever. A guard such as "stop and throw if `next_cursor` equals the cursor just sent" costs one line. Optional, because the server's cursor contract is covered by its own tests.

### 5. Preference: a negative numeric `Retry-After` falls through to `Date.parse`

`Retry-After: -5` fails the `seconds >= 0` check. It then goes to `Date.parse("-5")`, which V8 reads as a date in year −5, so the result is clamped to 0. The result is harmless, but the path is accidental. Returning `undefined` for any value that is a number but out of range would be clearer. Optional.

## Handbook

No handbook page changes in this PR. The SDK page for S25 is TICK-831.
