# Review: PR #956, TICK-752 (retire the JSONB scan and DataFilterTimeout)

- **Change:** [PR #956](https://github.com/Navaladi2019/TicketingSystem/pull/956), `TICK-752/remove-legacy-scan` into `TICK-740/search-finish`, reviewed at `93ebb609`.
- **Specs read:** ADR-006 D6 and D7, PLAN-007 D-7-6, the TICK-752 decisions of 2026-11-26 and 2026-11-27 (seed writes a flagged v1, `RegisterSeedFilterable` runs as `nava_app`), the TICK-1172 migration `20261127110000`, and `docs/delivery/decisions/2026-11-26-s22-retire-jsonb-scan-and-data-filter-timeout.md`.
- **CI:** [run 46044613739](https://github.com/Navaladi2019/TicketingSystem/actions/runs/46044613739) is green on `93ebb609`. The isolation report passes; the three rewritten I5 rows did real checks (30 fixed-column agreements, 467 `field_not_filterable`, 95 `masked_field`, 6 masked callers and 3 controls).
- **Verdict: APPROVE**, with the findings below. Finding 1 must be fixed before TICK-752 is done. Finding 2 must be fixed before the S22 kit (TICK-754) runs.

## What I checked

- The scan removal is complete. `DataFilter`, `Timeout`, `DataFilterTimeout`, `ErrStatementTimeout`, `boundDataFilter`, `timedOut`, `CountBudgetFor`, `parseLegacy`, `sameFilterValue`, the `r.data @>` term, `absentMatches`, the `dataTerm` mutant seam and `fieldindex.LegacyScan*` are gone from non-test code. The count budget is now always `CountBudget`.
- `resolveFilters` handles every registry state. Unflagged or `removing` gives 400 `field_not_filterable`, `backfilling` gives 422 `field_not_ready`, and `ready` sets `idx`. So `where()` never sees a data filter without a side table on the list route.
- `RegisterSeedFilterable` locks the definition row (`FOR UPDATE`, which is stronger than the `FOR SHARE` the Principal Software Engineer asked for) and requires `current_version = 1`, `published_by_system = 'onboarding'` and `published_by` NULL. Only `nava_onboard` can write that column, so an API-published type cannot pass the guard. Idempotency comes from `registerFilterable`, which skips rows that already exist.
- Onboarding order: the seed runs, then `RegisterSeedIndexes` for every template, then the admin group and `Activate`. An active tenant returns before the seed (`st.Active`, a replay).
- Tests on the risky paths, all run in CI: retry after a crash between the seed and registration (`TestS22OnboardingRetryAfterACrashBetweenSeedAndRegistration`), refusing a v2 and an API-published v1 (`TestS22RegisterSeedFilterableRefusesATypeOnboardingDidNotSeed`, `TestRegisterSeedFilterableRefusesWhatIsNotAnOnboardingSeed`), seed fields reaching `ready` through the real backfill (`TestS22SeedIndexesBecomeReady`), no activation when registration fails (`TestOnboardStopsBeforeActivationWhenIndexRegistrationFails`), and 400 for every type (`TestListFilterUnflaggedIsRefusedForEveryType`).

## Findings, most severe first

### 1. Breaks under an onboarding retry across the deploy: a tenant seeded before this change can never activate

`internal/entitytypes/entitytypes.go`, `RegisterSeedFilterable`.

- **Scenario:** a tenant's onboarding crashed after `SeedEntityTypes` on the old build. Its seeded v1 rows have `published_by_system` NULL; TICK-1172 added the column with no backfill. After this deploy the operator retries onboarding. `Seed` inserts nothing (`ON CONFLICT DO NOTHING`), and `RegisterSeedFilterable` sees `system == nil` and returns `ErrNotOnboardingSeed`. `Onboard` stops before `Activate` on every retry, so the tenant stays in `provisioning` until someone patches the data.
- **Fix (feature code, Senior Software Engineer):** compile v1 first and return nil when `c.Filterable()` is empty. The old seeds flag nothing, so there is nothing to register. Keep `ErrNotOnboardingSeed` for a v1 that flags fields and was not written by onboarding.
- **Test that catches it:** in `internal/db/dbtest/onboard_seed_index_test.go`, seed v1 the old way (unflagged schema, `published_by_system` NULL) while the tenant is provisioning, run `Onboard`, and assert the tenant becomes active with no registry rows.
- Today only a tenant stuck mid-onboarding at deploy time hits this. Fresh stacks and active tenants do not.

### 2. Bug in the S21 kit: earlier kits' data filters are not green on a fresh stack (criterion 3)

`docs/acceptance/guide/bruno/S21`. I found this by reading the kit; I have not run it.

- `S21-4` asserts that a fresh stack's incident type flags no property (`... .filter(p => p['x-nava-filterable']).length).to.eql(0)`). The incident seed now flags `priority`, so this fails.
- `S21-10` asserts that `priority` is still 422 after the writes (`priority still indexing after the writes`). On a fresh stack `priority` is registered at onboarding and backfilled by the worker before the kit's publish, so the list answers 200.
- `S21-8` flags `priority` again. That is harmless but no longer the step that indexes `priority`; only `due_at` is new there.
- **Fix:** update the S21 kit so `due_at` is the field that goes through backfilling and `priority` is ready from onboarding. Re-run it on a fresh stack in the S22 kit run (TICK-754). S10 (`S10-56`, `S10-60`) filters only on `description`, which is still unflagged, so it is unaffected.

### 3. Spec drift: the I5 data-filter row title promises a check the row no longer makes (accepted)

`internal/tenancy/isolation/i5_route_isolation_test.go`. The row title still says data-field eq/in agree with the Go check, but only fixed columns are compared. The Technical Lead accepted this until PR #931 restores the side-table oracle and retitles the row. TICK-752 stays open until #931 merges.

### 4. Spec drift: `filter_too_expensive` remains in the API spec but is never returned (deferred)

`problem.FilterTooExpensive` and its OpenAPI entry stay, though nothing returns them since `ErrStatementTimeout` is gone. The PR description defers this API change. It needs its own issue so it is not forgotten.

### 5. Preference: make `where()` fail closed on a data filter without a side table

`internal/server/records_list.go`, `(*listPlan).where`. A filter with `col == ""` and `idx == nil` now adds no term, so the filter would silently widen the list. `resolveFilters` makes that unreachable today, but any future caller of `planList` and `where` that skips `resolveFilters` (an export or count path, for example) would return unfiltered rows. Returning an error or panicking on that case would make it fail closed. Optional.

### Note for the handbook gate (Technical Lead)

Existing tenants lose `priority` and `risk` filters (400) until an admin publishes the flags and the backfill finishes. The PR documents this operator step. The S22 handbook pages should say it too.
