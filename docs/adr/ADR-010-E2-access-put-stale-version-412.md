---
type: adr-erratum
id: ADR-010-E2
title: "Erratum E2 to ADR-010: a stale expected version on PUT /entity-types/{key}/access is 412 version_conflict, not 409"
status: ACCEPTED
date: 2026-11-20
accepted: 2026-11-20 (Solution Architect verdict, TICK-1012, after Technical Architect critique comment 01a1f6bc-1688)
issue: TICK-1012
rfc: none (erratum from a handbook finding; RFC-022 stays archived as written)
supersedes: none
amends: "[[ADR-010-composable-access-rules]] 'API test scenarios', row PUT /api/v1/entity-types/{key}/access (the stale-version and double-PUT sentences); RFC-022 §5 'API test scenarios', bullets 3 and 4"
source: "TICK-945 (S30 handbook, PR #783), carried forward from TICK-939: docs/handbook/src/system/core-acl.md Notes"
evidence: "internal/server/access.go (PutEntityTypeAccess); internal/authz/publish.go (publishLocked); internal/server/access_integration_test.go (412 and replay tests); docs/acceptance/guide/bruno/S10/S10-14.bru"
deciders: "Solution Architect (author; verdict); Technical Architect (critique round 1, comment 01a1f6bc-1688: CHANGES, 1 blocking item, the in-flight 409, taken into the text; item 2 CRLF fixed; item 3 OpenAPI 409 routed to a follow-up); CTO (no tie-break needed, nothing contested)"
---

# ADR-010 erratum E2: a stale expected version is 412 `version_conflict`

> **Status: ACCEPTED** on 2026-11-20 by the Solution Architect (TICK-1012), under the merge policy set by vknavaladi. The Technical Architect's critique was CHANGES with one blocking item: a same-key duplicate while the first PUT is still running gets `409 idempotency_in_flight`, which the first text missed. It is fixed in this revision. Nothing was contested, so no CTO tie-break was needed.
>
> **Invariants source.** As in ADR-010: ADR-001 I1 to I9; the project description is empty, so the invariant check is against ADR-001 only. This erratum changes no behaviour and touches no invariant. I7 (idempotent side effects) is cited because the corrected double-PUT line describes it.

## In plain words

- **What was wrong:** ADR-010 says that saving access rules against an out-of-date version returns error code **409**. The running system returns **412**, as every other "only if the version is still X" route does.
- **What changes:** the ADR's test table is corrected to 412. No code changes.
- **What you'll see:** nothing. The route already behaves this way, and the S10 acceptance kit already checks it.

## What was wrong

The PUT route takes the version the admin last read in the `If-Match` header (the *expected version*). If someone else has published since, the save is refused. Two texts give the wrong status code for that refusal:

| Where | Text today |
|---|---|
| `ADR-010-composable-access-rules.md`, "API test scenarios", row `PUT /api/v1/entity-types/{key}/access` | "A stale expected version gives 409 with the current version and writes nothing. Two PUTs with the same expected version write once." |
| RFC-022 §5, "API test scenarios", bullets 3 and 4 | "PUT with a stale expected version → 409 `ConflictError` with the current version, nothing written." / "Two PUTs with the same expected version → the first writes, the second gets 409 (… I did not check whether the route also takes an idempotency key)." |

What the code does (`main` at `52d55150`):

1. `PutEntityTypeAccess` (`internal/server/access.go`) compares the `If-Match` versions with the newest published version. A mismatch is `*authz.ConflictError`, answered by `h.versionConflict`: **`412 version_conflict`** with the current version in the `ETag` header. Nothing is written.
2. Two concurrent PUTs that both pass that check are serialized by the entity-type lock in `publishLocked` (`internal/authz/publish.go`). The second re-checks the expected version under the lock, gets `ConflictError`, and so also gets **412**.
3. Every `/api/v1` write requires an `Idempotency-Key` (`internal/platform/idempotency`, ADR-001 D6 and I7). This route claims its key in a transaction of its own (only record create is a `ClaimInTx` route, `internal/server/api.go`). The same key with the same body, once the first request has completed, **replays** the first 200 (header `Idempotent-Replayed: true`) without running the handler again. If it arrives while the first is still running (the key's row is `in_flight` and within its 60 s `Lease`), it gets `409 idempotency_in_flight` with `Retry-After: 60`, and the handler does not run. The same key with a different body is `409 idempotency_key_reuse`. These two are the route's only 409s. Both are about the key, not the version.

409 `ConflictError` in the RFC was the Go error type's name read as a status code. `ConflictError` has always mapped to 412 on this route, as on schema publish and number formats. PLAN-011 does not repeat the 409.

## Correction

The ADR-010 test-table row and RFC-022 §5 bullets 3 and 4 read from this erratum on:

> A stale expected version (`If-Match` naming a version that is not the newest) gives **`412 version_conflict`** with the current version in the `ETag` header, and writes nothing. Two PUTs with the same `If-Match` write once: the second gets `412 version_conflict` with the first one's version as `ETag`, whether it arrives after the first or concurrently with it; if it reuses the first one's `Idempotency-Key` and body, it gets the first one's 200 replayed once the first has completed, or `409 idempotency_in_flight` (with `Retry-After`) while the first is still running. Either way, nothing more is written.

| # | Decision |
|---|---|
| **E2-1** | **The status is 412 `version_conflict`, with the current `ETag`.** This is what `access.go` returns today, what `TestPutAccessPublishesTheNextVersionWithItsAuditEntry` (`covers: putEntityTypeAccess 200 412 428`) and Bruno `S10-14.bru` assert, and what every `If-Match` route in the API returns. No code change. |
| **E2-2** | **The double-PUT line names both outcomes.** A second PUT with the same `If-Match` and a new key gets 412; with the same key and body it gets the replayed 200, or `409 idempotency_in_flight` while the first is still running. Either way one version and one audit entry exist, as `TestPutAccessReplayWithTheSameKeyWritesOneVersion` asserts. |
| **E2-3** | **The RFC stays archived as written.** `docs/adr/ADR-010/rfc/RFC-022-composable-access-rules.md` is not edited; this erratum is the correction, and the ADR-010 row points to it. |

**Not changed:** every ADR-010 decision (D1 to D13), the OpenAPI contract (it already lists 412 for this operation), the code, the tests and the acceptance kit. The OpenAPI operation does not list its two 409s; that pre-existing gap is a follow-up for the engineer, not part of this erratum.

## Competitive Gate (ADR-008)

**Result:** N/A. This corrects the documented status code to match shipped behaviour. It moves no product axis and changes no access decision.

## Rejected

| Option | Why |
|---|---|
| Change the route to return 409 for a stale version | Every `If-Match` route returns 412 (RFC 9110 §13.1.1: a failed precondition is 412). Making one route different breaks clients and the S10 kit to match a typo. |
| Edit RFC-022 in place | The archive records what was decided and why; corrections are recorded as errata so the history stays honest (ADR publication rule). |
| Leave the text and rely on the handbook note | The ADR is the normative text; engineers test against it. A wrong code there will be copied into a test. |

## Accepted risk

None. Documentation matches shipped and tested behaviour.

## Evidence

- `internal/server/access.go`: `PutEntityTypeAccess`, the `errors.As(err, &conflict)` case → `h.versionConflict(ctx, int64(conflict.Current))`.
- `internal/authz/publish.go`: `publishLocked` re-checks `expected` under `lockDefinitionSQL` and returns `&ConflictError{Current: have}`.
- `internal/server/access_integration_test.go`: `covers: putEntityTypeAccess 200 412 428`; the stale-`If-Match` case asserts 412 and the current `ETag`; `TestPutAccessReplayWithTheSameKeyWritesOneVersion` asserts the replay writes one version and one audit entry, and the reused key with another body is `409 idempotency_key_reuse`.
- `internal/platform/idempotency/idempotency.go`: a route outside `ClaimInTx` answers a concurrent duplicate of a young `in_flight` key with `409 idempotency_in_flight` and `Retry-After` = `Lease` (60 s).
- `docs/acceptance/guide/bruno/S10/S10-14.bru`: "PUT with a stale If-Match: 412 and the current ETag".
