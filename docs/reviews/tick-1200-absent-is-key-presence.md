# Review: TICK-1200, D12 window "absent" is key presence (PR #931, commit 446bd265)

Reviewer: Quality, Testing & CI Advisor. Specs: ADR-009 D12, ADR-006 D6, TICK-1200 scope.

## CI

Full run on the branch head (manual dispatch, because the PR conflicts with its base
and so `pull_request` did not start): run 46074574240, **success**, every job.

- `TestI5Agreement` (the I5 "data filters" row, red in run 46067368986 on the
  `prio: null` cases): PASS. The oracle was not touched.
- `TestListFilterWindowDoesNotReturnAStoredNullOrANonCoercibleValue` (integration): PASS.
- `TestRenderAbsentIsKeyPresenceNotNoSideRow` (unit): PASS.
- `TestS22AbsentSiteAllowlist` and the real-repo attack 4 scan (isolation): PASS.
- `TestListFilterUnionShapePlanReadsTheQueueAndNoSeqScanOnRecords`: PASS, so the plan is not affected.

## Findings, most severe first

1. **Spec drift (must fix before merge).** PR #931 conflicts with
   `TICK-740/search-finish` (`docs/handbook/assets/search-index.js`). #982
   (TICK-929) has landed on the base, and its `docs/handbook/src/database/records.md`
   lines 83-85 say the opposite of this fix: "'Lacks the field' means no side-table
   row for it, so a stored value that does not coerce also counts as absent while
   the window is open." After the merge, the handbook would contradict the code.
   Fix: merge the base, rewrite that paragraph to key presence (a stored `null` or a
   non-coercible value is not absent), add the new integration test to that page's
   `tests:` front matter, keep or drop the schema-backfill note so the two pages
   agree, and rebuild the handbook. The rebuild also regenerates `search-index.js`.
   This is the page that TICK-1200 scope item 4 names.
2. **Missing test (minor).** In `records_list_backfill_test.go`,
   `TestRenderAbsentIsKeyPresenceNotNoSideRow` checks that the side-row term is gone
   with `X || A && B`, where `A` pins `v.field_id = $3::uuid)`. That placeholder
   number need not match the old rendering, so the negative check can pass
   vacuously. The integration test covers the behaviour, so this is not a blocker.
   A plain `!strings.Contains(sql, "NOT EXISTS (SELECT 1 FROM rfv_")` would make the
   check real.
3. **Preference.** The attack 4 exemption checks the function and the fragment, not
   what follows the fragment. A key spliced into the SQL
   (`"NOT (r.data ? '" + k + "')"`) would pass the scan. Today the unit test's
   bound-parameter check catches that, so this is acceptable as it stands.

## Attack 4 allowlist: approved

- The exemption holds only for string literals inside the `fieldAbsent` FuncDecl body
  of `internal/server/records_list_index.go`. It strips one occurrence of the exact
  fragment `NOT (r.data ? `, and the rest of the literal is still scanned.
- `TestS22AbsentSiteAllowlist` covers the allowed site and four red cases: the same
  fragment in another function, the same fragment in another file, another operator
  in `fieldAbsent`, and a second operator beside the fragment. This meets the
  TICK-1200 criterion.
- Isolation: the term reads the row's own `data` with a key bound from the compiled
  schema, and it adds no cross-row or cross-tenant read. I see no concern to route
  to the Security & Multi-Tenancy Reviewer.

## Verdict

Code: approved; CI green. Merge is blocked only on finding 1: merge the base, fix
the records.md paragraph, rebuild the handbook. Then read the PR's CI once.
