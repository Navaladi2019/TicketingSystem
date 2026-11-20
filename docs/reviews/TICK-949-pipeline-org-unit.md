# Review: TICK-949, the write pipeline sets `records.org_unit_id` on create

- **PR:** #801 (`TICK-949/pipeline-org-unit-on-create` → `TICK-946/in-my-region`)
- **Spec:** ADR-010 D8 and I4; PLAN-011 P11-16
- **Reviewer:** Quality, Testing & CI Advisor
- **Verdict:** APPROVED at `48add97`. The GitHub `ci` workflow is green: run 44771892167, with `unit`, `integration`, `lint`, `stack`, `coverage` and `full-suite` all passing.

## Round 1 (`8d3c0c3`): changes requested, CI red

| # | Severity | Finding | Resolution in `48add97` |
|---|---|---|---|
| 1 | Bug | `PageStatement` and `branchedPage` in `internal/records/store/list.go` did not select `org_unit_id`, but share `scan`. Every record list call returned 500 ("got 19 and 20"). | Both list SELECTs read `r.org_unit_id`. The `TestList*` integration tests pass. |
| 2 | Breaks the round-trip budget (TICK-648) | The `PrimaryOrgUnit` lookup was a separate round trip, so create took 5 trips instead of 4. | The lookup is a subquery in the payload-preview statement of `checkCreate`'s batch. `TestRecordCreateAndGetRoundTrips` and `TestCreateRecordFailureInFirstOrLastTripLeavesNothing` pass with the budget unchanged. |
| 3 | Bug (test) | The replay test sent an idempotency key on a user command, which only system writers may use. | Uses a system-source workflow command. |
| 4 | Missing test | The gapless and at-commit insert paths were not checked for `org_unit_id`. | `TestOrgUnitOnEveryInsertPath` covers the gaps, gapless and gaps-at-commit paths. Each case checks the stored unit and the audit entry. |
| 5 | Preference | The FK test accepted any error. | Requires SQLSTATE `23503`. |
| 6 | Preference | A data property named `org_unit_id` would be overwritten in the audit payload. | `metadata.Compile` refuses that property name. Covered by a `TestCompileRefuses` case. |

## Round 2 (`48add97`): notes, none blocking

- **Design note (for the "in my region" tasks).** The org unit is now resolved in the `checkCreate` batch, which runs after `canWrite`. That is correct today, because `authz.Row` has no org unit. A later task that adds an `org_unit_id` term to the row-scope write check must move the lookup before `canWrite`, for example into `queuePrefetch`. Otherwise the create write check will see a NULL unit.
- **Same-row guarantee.** The INSERT stores `r.OrgUnitID`, which is scanned from the same statement that built the audit preview. The insert paths then compare the stored payload with the preview, so the audit entry and the stored row cannot disagree.
- **Uniqueness.** The scalar subquery relies on the one-primary-per-principal index on `org_unit_members`. If that index were dropped, a create would fail with "more than one row returned by a subquery".

## Coverage of the issue's scenarios

| Scenario | Test |
|---|---|
| Create by a requester with a primary unit | `TestCreateSetsOrgUnitFromRequestersPrimaryUnit` |
| Create by a requester with none | `TestCreateWithoutPrimaryUnitLeavesOrgUnitNull` |
| `org_unit_id` in the body is refused and nothing is written (create and update) | `TestCreateRefusesOrgUnitInBody` |
| An idempotent replay creates one record | `TestOrgUnitIdempotentReplayOneRecord` |
| The requester moves unit afterwards | `TestOrgUnitStaysWhenRequesterMoves` |
| Another tenant's unit or membership | `TestOrgUnitNeverReadsAnotherTenantsMembership` |
| Every insert path | `TestOrgUnitOnEveryInsertPath` |

## Handbook

There is no DDL. The audit payload now carries `org_unit_id`, and the schema compiler reserves the property name `org_unit_id`. The Technical Lead should reflect both in the write-pipeline handbook page and in `docs/data/org-units.md`.
