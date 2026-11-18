## Review: TICK-746 (PLAN-007 P7-06), list filter on data fields through the side tables

- **PR:** #709, `TICK-746/list-filter-exists` → `TICK-739/filterable-fields`, reviewed at head `d456c4d0`
- **Specs read:** the TICK-746 issue (D6 order, operators, `EXISTS` shape, D-7-6 and D-7-14 legacy allow-list), ADR-006 D6/D7, the D25 decision on the data-filter timeout
- **CI:** run `44330791514` is all green, including `coverage` and `full-suite`
- **Verdict:** CHANGES REQUESTED, because of one Bug: a regression in an accepted acceptance kit

## Findings

### 1. Bug: S10.60 in the accepted S10 acceptance kit now answers 400

- **What breaks:** `docs/acceptance/guide/bruno/S10/S10-60.bru` sends `GET /records/incident?filter[description]=…` as the admin and expects 200 with exactly the `/service-desk` incident. S10 was accepted on that evidence (`docs/acceptance/s10-access.md`, C4: "S10.60: the admin can filter on it").
- **Why:** this PR sends every unflagged data field that is not in `fieldindex.LegacyScanFields` to 400 `field_not_filterable`. `incident.description` is neither flagged nor in the allow-list.
- **What CI misses:** CI does not run the Bruno kit, so it stays green. The Go S10 tests were changed to flag the fields they filter on, but the kit was not.
- **Expected:** every request in the S10 kit still passes on a fresh stack, or the kit and its acceptance record are changed on purpose.
- **Actual:** S10.60 fails with 400 `field_not_filterable`.
- **Fix options:**
  - **(a)** In the S10 kit setup, flag `description` as `x-nava-filterable` on the seeded incident type and wait for `ready` before S10.60. S10.56, the masked case, keeps answering `masked_field`, because the masked check still comes first. This keeps C4's evidence meaningful.
  - **(b)** Change S10.60 to expect 400 `field_not_filterable` and record the change in `s10-access.md`. C4 then loses its "admin can filter" evidence.
  - Do not add `incident.description` to `LegacyScanFields`: the spec says "Nothing is added to that list".
  - (a) is the better choice. The Technical Lead owns the kit wording.
- **Test that catches it:** run the S10 Bruno folder against a fresh stack built from this branch.

### 2. Preference: when two data filters fail, the first one by field name wins, not the strictest error

`resolveFilters` (`internal/server/records_list_index.go:75-117`) checks the data filters in sorted key order and returns the first refusal. For example, `filter[a]` on a backfilling field and `filter[b]` on an unflagged field answer 422 for `a`, not 400 for `b`. The result is deterministic, because the keys are sorted in `planList`, and it doesn't leak anything. Running D6 step 2 for every field before step 3 would make the 400 always win, if the spec owner wants that. It is optional.

### 3. Preference: the `acl` label of the latency histogram is guessed from the SQL text

`ListRecords` sets `narrowing := branches != nil || len(readArgs) > 0 || strings.Contains(readSQL, "r.")`. It only feeds a metric label, but a future change to the ACL compiler's SQL could flip it without anyone noticing. Prefer a flag returned by the ACL compiler.

## Checked and correct

- **D6 order per field:** masked (`checkPlan`, before the registry is read), then not flagged (`removing` counts as not flagged), then `backfilling` (422 with `index_progress` held to 0–100), then operator and value. Covered by `TestListFilterRefusalsFollowTheD6Order` and `TestListFilterBackfillingAnswers422WithOnlyAPercentage`.
- **Operators:** `lt`/`lte`/`gt`/`gte` work only on `num` and `ts`. `in` takes at most 100 values. Value parsing follows `record_field_coerce`: finite double, `true`/`false`, RFC 3339 with an offset or a day, text up to 2,000 bytes. Covered by `TestCoerceFilterValue` and `TestListFilterValueErrorsAreValidationFailed`.
- **`EXISTS` shape:** the table name comes from the fixed `SideTable()` set and every value is a bind parameter. It is correlated on `tenant_id` and `record_id`, uses `COLLATE "C"` on text, and has no extra count query. Covered by `TestExistsTermShape`, `TestListFilterPushdownIsAnIndexCond` (CI-1, as `nava_app`) and `TestD7SideTablesAreReadOnlyInTheListEXISTS`.
- **Mixed fixed-column and data filters share placeholder numbering correctly.** The oracle test covers `state` with `qty`, `name` and `vip`.
- **Row ACL (CI-6c), cross-tenant, refusals that write nothing, and the legacy allow-list keyed by (type, field):** each has its own integration test.
- **Registry read failure** answers 500 and never falls back to the legacy scan (`TestListFilterRegistryReadFailureIsA500`). The scan and `rows.Err()` paths are unit-tested through `collectIndexFields`.
- **Legacy-scan timeout:** it now applies only to legacy-scan lists (`hasLegacyScan`), as D-7-6 says. Indexed lists run without the 2-second data-filter timeout, as the spec intends.

## Routed elsewhere

- **Isolation:** none. The `EXISTS` correlates on `r.tenant_id` and the cross-tenant test covers it.
- **Handbook:** the list/filter API page and the ADR-006 page change. This goes to the Technical Lead's handbook task, as the PR description says.
- **Flaky test:** `TestAllocatorPoolDoesNotStarveThePool` failed once on this PR and is unrelated. It is tracked in TICK-910.
